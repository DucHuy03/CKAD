package handlers

import (
	"database/sql"
	"net/http"

	"movie-service/internal/models"
)

// ShowtimeHandler là handler quan trọng nhất đối với booking-service:
// GET /showtimes/{id} sẽ được booking-service gọi (qua ambassador nginx của
// chính booking-service, proxy sang movie-service) để lấy giá vé và
// tổng số ghế trước khi cho user hold ghế.
type ShowtimeHandler struct {
	db *sql.DB
}

func NewShowtimeHandler(db *sql.DB) *ShowtimeHandler {
	return &ShowtimeHandler{db: db}
}

// Create xử lý POST /showtimes.
func (h *ShowtimeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var s models.Showtime
	if !decodeJSON(w, r, &s) {
		return
	}

	err := h.db.QueryRowContext(r.Context(), `
		INSERT INTO showtimes (movie_id, cinema_id, room_name, start_time, end_time, price, total_seats)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`, s.MovieID, s.CinemaID, s.RoomName, s.StartTime, s.EndTime, s.Price, s.TotalSeats,
	).Scan(&s.ID, &s.CreatedAt)
	if err != nil {
		// Loi thuong gap: movie_id/cinema_id khong ton tai -> vi pham foreign key.
		// Postgres se tra ve loi ro rang, khong can tu code check truoc (giam 1 round-trip DB).
		writeError(w, http.StatusBadRequest, "tao showtime that bai (kiem tra movie_id/cinema_id co ton tai khong): "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, s)
}

// List xử lý GET /showtimes, hỗ trợ lọc qua query param ?movie_id=...&cinema_id=...
// Ví dụ: GET /showtimes?movie_id=<uuid> -> tất cả suất chiếu của 1 phim,
// dùng khi user chọn phim rồi cần xem các suất chiếu khả dụng.
func (h *ShowtimeHandler) List(w http.ResponseWriter, r *http.Request) {
	movieID := r.URL.Query().Get("movie_id")
	cinemaID := r.URL.Query().Get("cinema_id")

	// Xay query dong theo filter duoc truyen vao, van dung parameterized query
	// ($1, $2...) de tranh SQL injection du filter la optional.
	query := `
		SELECT id, movie_id, cinema_id, room_name, start_time, end_time, price, total_seats, created_at
		FROM showtimes
		WHERE ($1 = '' OR movie_id::text = $1)
		  AND ($2 = '' OR cinema_id::text = $2)
		ORDER BY start_time ASC
	`
	rows, err := h.db.QueryContext(r.Context(), query, movieID, cinemaID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "truy van danh sach showtime that bai: "+err.Error())
		return
	}
	defer rows.Close()

	showtimes := []models.Showtime{}
	for rows.Next() {
		var s models.Showtime
		if err := rows.Scan(&s.ID, &s.MovieID, &s.CinemaID, &s.RoomName,
			&s.StartTime, &s.EndTime, &s.Price, &s.TotalSeats, &s.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "doc du lieu showtime that bai: "+err.Error())
			return
		}
		showtimes = append(showtimes, s)
	}

	writeJSON(w, http.StatusOK, showtimes)
}

// GetByID xử lý GET /showtimes/{id} - endpoint booking-service sẽ gọi nhiều nhất.
func (h *ShowtimeHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var s models.Showtime
	err := h.db.QueryRowContext(r.Context(), `
		SELECT id, movie_id, cinema_id, room_name, start_time, end_time, price, total_seats, created_at
		FROM showtimes WHERE id = $1
	`, id).Scan(&s.ID, &s.MovieID, &s.CinemaID, &s.RoomName,
		&s.StartTime, &s.EndTime, &s.Price, &s.TotalSeats, &s.CreatedAt)

	if isNoRows(err) {
		writeError(w, http.StatusNotFound, "khong tim thay showtime id="+id)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "truy van showtime that bai: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, s)
}
