package handlers

import (
	"context"
	"database/sql"
	"math/rand"
	"net/http"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"payment-service/internal/bookingclient"
	"payment-service/internal/models"
	"payment-service/internal/rabbitmq"
)

// PaymentHandler xử lý toàn bộ luồng thanh toán.
type PaymentHandler struct {
	db            *sql.DB
	ch            *amqp.Channel
	bookingClient *bookingclient.Client
}

func NewPaymentHandler(db *sql.DB, ch *amqp.Channel, bookingClient *bookingclient.Client) *PaymentHandler {
	return &PaymentHandler{db: db, ch: ch, bookingClient: bookingClient}
}

// Process xử lý POST /payments/process - đây là endpoint trung tâm nối liền
// booking-service (confirm) và notification-service (qua event RabbitMQ).
//
// FLOW:
//  1. Insert 1 dòng payment status=PENDING (ghi nhận "đã có yêu cầu thanh toán")
//  2. Giả lập gọi payment gateway thật (simulatePaymentGateway) - vì đây là
//     bài lab học k8s, không tích hợp Stripe/VNPay thật, chỉ mô phỏng
//     thành công/thất bại theo tỉ lệ ngẫu nhiên để có đủ 2 nhánh luồng
//     nghiệp vụ (thành công -> confirm booking, thất bại -> dừng lại)
//     3a. Nếu thành công: UPDATE status=SUCCESS -> gọi booking-service confirm
//     -> publish event "payment.success" lên RabbitMQ cho notification-service
//     3b. Nếu thất bại: UPDATE status=FAILED kèm lý do, KHÔNG gọi booking-service
//     (booking vẫn ở trạng thái HOLD, sẽ tự hết hạn theo TTL nếu user
//     không thử thanh toán lại kịp - xem CronJob cleanup ở booking-service)
func (h *PaymentHandler) Process(w http.ResponseWriter, r *http.Request) {
	var req models.ProcessRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.BookingID == "" || req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "booking_id va amount phai hop le")
		return
	}
	if req.Method == "" {
		req.Method = "CARD"
	}

	ctx := r.Context()

	// Buoc 1: ghi nhan yeu cau thanh toan o trang thai PENDING truoc
	var payment models.Payment
	err := h.db.QueryRowContext(ctx, `
		INSERT INTO payments (booking_id, showtime_id, user_id, amount, method, status)
		VALUES ($1, $2, $3, $4, $5, 'PENDING')
		RETURNING id, booking_id, showtime_id, user_id, amount, method, status, created_at, updated_at
	`, req.BookingID, req.ShowtimeID, req.UserID, req.Amount, req.Method,
	).Scan(&payment.ID, &payment.BookingID, &payment.ShowtimeID, &payment.UserID,
		&payment.Amount, &payment.Method, &payment.Status, &payment.CreatedAt, &payment.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "tao payment record that bai: "+err.Error())
		return
	}

	// Buoc 2: gia lap goi payment gateway
	success, failureReason := simulatePaymentGateway(req)

	if !success {
		h.updatePaymentStatus(ctx, payment.ID, models.StatusFailed, failureReason)
		payment.Status = models.StatusFailed
		payment.FailureReason = failureReason
		writeJSON(w, http.StatusPaymentRequired, payment) // 402 Payment Required
		return
	}

	// Buoc 3a: thanh toan thanh cong -> confirm booking ben booking-service
	if err := h.bookingClient.ConfirmBooking(ctx, req.BookingID); err != nil {
		// Thanh toan "thanh cong" nhung khong confirm duoc booking (vd booking
		// da het han TTL dung luc nay) - day la tinh huong can xu ly that can
		// than trong he thong that (hoan tien...). O muc bai lab, ta ghi nhan
		// payment la FAILED kem ly do ro rang de de debug, khong publish event.
		reason := "thanh toan thanh cong nhung confirm booking that bai: " + err.Error()
		h.updatePaymentStatus(ctx, payment.ID, models.StatusFailed, reason)
		payment.Status = models.StatusFailed
		payment.FailureReason = reason
		writeError(w, http.StatusConflict, reason)
		return
	}

	h.updatePaymentStatus(ctx, payment.ID, models.StatusSuccess, "")
	payment.Status = models.StatusSuccess

	// Buoc 3b: publish event cho notification-service - lam "best effort",
	// khong lam that bai ca request neu publish loi (thanh toan + confirm da
	// xong xuoi, khong nen tra loi cho user chi vi RabbitMQ tam thoi truc trac;
	// trade-off nay can luu y: notification co the bi mat trong truong hop nay,
	// he thong that nen dung outbox pattern de dam bao at-least-once).
	event := rabbitmq.PaymentSuccessEvent{
		PaymentID:  payment.ID,
		BookingID:  req.BookingID,
		ShowtimeID: req.ShowtimeID,
		UserID:     req.UserID,
		Amount:     req.Amount,
		Seats:      req.Seats,
		PaidAt:     time.Now().Format(time.RFC3339),
	}
	if err := rabbitmq.PublishPaymentSuccess(ctx, h.ch, event); err != nil {
		// Chi log, khong tra loi that bai cho client - xem giai thich o comment tren.
		writeJSON(w, http.StatusOK, payment)
		return
	}

	writeJSON(w, http.StatusOK, payment)
}

// GetByID xử lý GET /payments/{id}.
func (h *PaymentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var p models.Payment
	err := h.db.QueryRowContext(r.Context(), `
		SELECT id, booking_id, showtime_id, user_id, amount, method, status, failure_reason, created_at, updated_at
		FROM payments WHERE id = $1
	`, id).Scan(&p.ID, &p.BookingID, &p.ShowtimeID, &p.UserID, &p.Amount,
		&p.Method, &p.Status, &p.FailureReason, &p.CreatedAt, &p.UpdatedAt)

	if isNoRows(err) {
		writeError(w, http.StatusNotFound, "khong tim thay payment id="+id)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "truy van payment that bai: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, p)
}

func (h *PaymentHandler) updatePaymentStatus(ctx context.Context, id string, status models.PaymentStatus, reason string) {
	h.db.ExecContext(ctx, `
		UPDATE payments SET status = $1, failure_reason = $2, updated_at = now() WHERE id = $3
	`, status, reason, id)
}

// simulatePaymentGateway giả lập việc gọi 1 cổng thanh toán thật (Stripe,
// VNPay, Momo...). KHÔNG dùng cho production - đây chỉ là mock để bài lab
// có đủ 2 nhánh luồng (thành công/thất bại) mà không cần tích hợp thật.
//
// Quy tắc giả lập: method "FAIL_TEST" luôn thất bại (dùng để test có chủ
// đích nhánh lỗi khi thực hành lab), còn lại thành công với xác suất 90%.
func simulatePaymentGateway(req models.ProcessRequest) (success bool, failureReason string) {
	if req.Method == "FAIL_TEST" {
		return false, "gia lap that bai theo yeu cau (method=FAIL_TEST)"
	}
	if rand.Float64() < 0.9 {
		return true, ""
	}
	return false, "gia lap: cong thanh toan tu choi giao dich"
}
