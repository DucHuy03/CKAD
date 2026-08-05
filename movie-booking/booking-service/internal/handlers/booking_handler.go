package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"booking-service/internal/config"
	"booking-service/internal/models"
	"booking-service/internal/movieclient"
)

// BookingHandler chứa logic nghiệp vụ quan trọng nhất hệ thống: giữ ghế,
// xác nhận, huỷ. Đây cũng là nơi minh hoạ 1 race condition kinh điển:
// "2 user cùng bấm chọn ghế A7 trong cùng 1 khoảnh khắc - ai thắng?"
type BookingHandler struct {
	db          *sql.DB
	redisClient *redis.Client
	movieClient *movieclient.Client
	holdTTL     time.Duration
}

func NewBookingHandler(db *sql.DB, redisClient *redis.Client, movieClient *movieclient.Client, cfg config.Config) *BookingHandler {
	return &BookingHandler{
		db:          db,
		redisClient: redisClient,
		movieClient: movieClient,
		holdTTL:     cfg.HoldTTL,
	}
}

// --- Quy uoc dat ten Redis key, dat thanh ham de khong go sai chuoi format o nhieu noi ---

// holdKey: 1 key rieng cho MOI GHE, co TTL - day la co che tu dong het han.
// Vi du: "hold:9d1e...:A7" -> gia tri la booking_id dang giu ghe do.
func holdKey(showtimeID, seat string) string {
	return fmt.Sprintf("hold:%s:%s", showtimeID, seat)
}

// bookedSetKey: 1 SET dung chung cho ca showtime, luu cac ghe DA BAN
// (khong co TTL - ghe da ban thi ban vinh vien cho suat chieu do).
func bookedSetKey(showtimeID string) string {
	return fmt.Sprintf("booked:%s", showtimeID)
}

// Hold xử lý POST /bookings/hold.
//
// FLOW (đọc kỹ vì đây là logic quan trọng nhất service):
//  1. Gọi movie-service lấy thông tin showtime (giá vé, tồn tại hay không)
//  2. Với TỪNG ghế trong danh sách yêu cầu:
//     a. Check ghế đã nằm trong tập "đã bán" (bookedSetKey) chưa -> nếu có, từ chối ngay
//     b. Thử SETNX (Set-if-Not-eXists) key hold cho ghế đó kèm TTL
//     - SETNX là lệnh ATOMIC ở Redis: 2 request đến cùng lúc, chỉ 1 request
//     thắng (key chưa tồn tại -> set thành công), request kia luôn nhận
//     "đã tồn tại" -> đây chính là cách giải quyết race condition "2 user
//     cùng bấm 1 ghế" mà không cần lock kiểu mutex ở tầng ứng dụng.
//  3. Nếu 1 ghế bất kỳ bị fail ở bước 2b (đã có người khác giữ) -> ROLLBACK
//     các ghế đã set thành công trước đó trong CÙNG request này, trả lỗi 409.
//     (Lưu ý: đây là "atomic theo từng key", không phải atomic toàn bộ tập
//     ghế - vẫn có khe hở lý thuyết nếu muốn atomic tuyệt đối phải dùng Lua
//     script (EVAL) để Redis xử lý tất cả trong 1 lệnh duy nhất. Ở mức bài
//     lab, rollback thủ công là đủ để hiểu nguyên lý.)
//  4. Insert booking status=HOLD vào Postgres, kèm expires_at.
func (h *BookingHandler) Hold(w http.ResponseWriter, r *http.Request) {
	var req models.HoldRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.ShowtimeID == "" || len(req.Seats) == 0 {
		writeError(w, http.StatusBadRequest, "showtime_id va seats khong duoc rong")
		return
	}

	ctx := r.Context()

	// Buoc 1: xac nhan showtime ton tai + lay gia ve tu movie-service
	showtime, err := h.movieClient.GetShowtime(ctx, req.ShowtimeID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "khong lay duoc thong tin showtime: "+err.Error())
		return
	}

	// Buoc 2: thu giu tung ghe, rollback neu co ghe nao that bai
	heldSeats := make([]string, 0, len(req.Seats))
	bookingIDPlaceholder := "pending" // dung tam, se thay bang id that sau khi insert Postgres

	for _, seat := range req.Seats {
		// 2a. Ghe da ban vinh vien chua?
		alreadyBooked, err := h.redisClient.SIsMember(ctx, bookedSetKey(req.ShowtimeID), seat).Result()
		if err != nil {
			h.rollbackHolds(ctx, req.ShowtimeID, heldSeats)
			writeError(w, http.StatusInternalServerError, "loi kiem tra redis: "+err.Error())
			return
		}
		if alreadyBooked {
			h.rollbackHolds(ctx, req.ShowtimeID, heldSeats)
			writeError(w, http.StatusConflict, fmt.Sprintf("ghe %q da duoc ban", seat))
			return
		}

		// 2b. SETNX: chi thanh cong neu key CHUA ton tai (chua ai giu ghe nay).
		ok, err := h.redisClient.SetNX(ctx, holdKey(req.ShowtimeID, seat), bookingIDPlaceholder, h.holdTTL).Result()
		if err != nil {
			h.rollbackHolds(ctx, req.ShowtimeID, heldSeats)
			writeError(w, http.StatusInternalServerError, "loi giu ghe qua redis: "+err.Error())
			return
		}
		if !ok {
			// Ghe nay dang bi nguoi khac giu (con TTL) -> rollback nhung ghe da giu duoc, tra 409.
			h.rollbackHolds(ctx, req.ShowtimeID, heldSeats)
			writeError(w, http.StatusConflict, fmt.Sprintf("ghe %q dang duoc nguoi khac giu, thu lai sau", seat))
			return
		}
		heldSeats = append(heldSeats, seat)
	}

	// Buoc 3: da giu duoc het tat ca ghe -> ghi booking vao Postgres
	totalPrice := showtime.Price * float64(len(req.Seats))
	expiresAt := time.Now().Add(h.holdTTL)

	var booking models.Booking
	err = h.db.QueryRowContext(ctx, `
		INSERT INTO bookings (showtime_id, user_id, seats, total_price, status, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, showtime_id, user_id, seats, total_price, status, expires_at, created_at, updated_at
	`, req.ShowtimeID, req.UserID, pq.Array(req.Seats), totalPrice, models.StatusHold, expiresAt,
	).Scan(&booking.ID, &booking.ShowtimeID, &booking.UserID, &booking.Seats,
		&booking.TotalPrice, &booking.Status, &booking.ExpiresAt, &booking.CreatedAt, &booking.UpdatedAt)
	if err != nil {
		// Insert Postgres that bai sau khi da giu ghe thanh cong o Redis ->
		// phai rollback Redis, neu khong ghe se bi "khoa oan" cho toi khi TTL het han.
		h.rollbackHolds(ctx, req.ShowtimeID, heldSeats)
		writeError(w, http.StatusInternalServerError, "luu booking that bai: "+err.Error())
		return
	}

	// Buoc 4: cap nhat lai gia tri that cua hold key tu "pending" thanh booking.ID that,
	// de sau nay Confirm/Cancel biet booking nao dang giu ghe nao.
	for _, seat := range req.Seats {
		h.redisClient.Set(ctx, holdKey(req.ShowtimeID, seat), booking.ID, h.holdTTL)
	}

	writeJSON(w, http.StatusCreated, booking)
}

// rollbackHolds xoá các hold key đã set thành công trước khi gặp lỗi giữa chừng -
// tránh để ghế bị "khoá oan" cho tới khi TTL tự hết hạn (làm booking-service
// từ chối nhầm những user khác trong lúc chờ TTL, dù bản thân giao dịch này đã fail).
func (h *BookingHandler) rollbackHolds(ctx context.Context, showtimeID string, seats []string) {
	for _, seat := range seats {
		h.redisClient.Del(ctx, holdKey(showtimeID, seat))
	}
}

// Confirm xử lý POST /bookings/{id}/confirm - được payment-service gọi
// SAU KHI thanh toán thành công.
//
// FLOW: kiểm tra booking đang ở trạng thái HOLD và chưa hết hạn -> chuyển
// ghế từ "đang giữ tạm" (hold key có TTL) sang "đã bán vĩnh viễn" (thêm vào
// booked set, xoá hold key) -> cập nhật status Postgres thành CONFIRMED.
func (h *BookingHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()

	booking, err := h.loadBooking(ctx, id)
	if isNoRows(err) {
		writeError(w, http.StatusNotFound, "khong tim thay booking id="+id)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "truy van booking that bai: "+err.Error())
		return
	}

	if booking.Status != models.StatusHold {
		writeError(w, http.StatusConflict, fmt.Sprintf("booking dang o trang thai %q, khong the confirm", booking.Status))
		return
	}
	if booking.ExpiresAt != nil && time.Now().After(*booking.ExpiresAt) {
		// Da het han nhung CronJob chua kip quet - tu danh dau EXPIRED ngay tai day.
		h.updateStatus(ctx, id, models.StatusExpired)
		writeError(w, http.StatusConflict, "booking da het han giu ghe")
		return
	}

	// Chuyen tung ghe tu "hold" sang "booked vinh vien"
	for _, seat := range booking.Seats {
		h.redisClient.SAdd(ctx, bookedSetKey(booking.ShowtimeID), seat)
		h.redisClient.Del(ctx, holdKey(booking.ShowtimeID, seat))
	}

	if err := h.updateStatus(ctx, id, models.StatusConfirmed); err != nil {
		writeError(w, http.StatusInternalServerError, "cap nhat trang thai booking that bai: "+err.Error())
		return
	}

	booking.Status = models.StatusConfirmed
	writeJSON(w, http.StatusOK, booking)
}

// Cancel xử lý POST /bookings/{id}/cancel - user chủ động huỷ trước khi hết hạn.
func (h *BookingHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()

	booking, err := h.loadBooking(ctx, id)
	if isNoRows(err) {
		writeError(w, http.StatusNotFound, "khong tim thay booking id="+id)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "truy van booking that bai: "+err.Error())
		return
	}
	if booking.Status != models.StatusHold {
		writeError(w, http.StatusConflict, fmt.Sprintf("booking dang o trang thai %q, khong the huy", booking.Status))
		return
	}

	h.rollbackHolds(ctx, booking.ShowtimeID, booking.Seats)

	if err := h.updateStatus(ctx, id, models.StatusCancelled); err != nil {
		writeError(w, http.StatusInternalServerError, "cap nhat trang thai booking that bai: "+err.Error())
		return
	}

	booking.Status = models.StatusCancelled
	writeJSON(w, http.StatusOK, booking)
}

// GetByID xử lý GET /bookings/{id}.
func (h *BookingHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	booking, err := h.loadBooking(r.Context(), id)
	if isNoRows(err) {
		writeError(w, http.StatusNotFound, "khong tim thay booking id="+id)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "truy van booking that bai: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, booking)
}

// loadBooking là helper nội bộ dùng chung cho Confirm/Cancel/GetByID.
func (h *BookingHandler) loadBooking(ctx context.Context, id string) (models.Booking, error) {
	var b models.Booking
	err := h.db.QueryRowContext(ctx, `
		SELECT id, showtime_id, user_id, seats, total_price, status, expires_at, created_at, updated_at
		FROM bookings WHERE id = $1
	`, id).Scan(&b.ID, &b.ShowtimeID, &b.UserID, &b.Seats,
		&b.TotalPrice, &b.Status, &b.ExpiresAt, &b.CreatedAt, &b.UpdatedAt)
	return b, err
}

// updateStatus là helper nội bộ cập nhật status + updated_at.
func (h *BookingHandler) updateStatus(ctx context.Context, id string, status models.BookingStatus) error {
	_, err := h.db.ExecContext(ctx, `
		UPDATE bookings SET status = $1, updated_at = now() WHERE id = $2
	`, status, id)
	return err
}
