package handlers

import (
	"database/sql"
	"net/http"

	"movie-service/internal/models"
)

// CinemaHandler theo đúng pattern của MovieHandler (xem comment chi tiết ở
// movie_handler.go). Ở đây chỉ làm Create/List/GetByID - đủ dùng cho
// booking-service tra cứu thông tin rạp, không cần Update/Delete phức tạp
// trong phạm vi bài lab.
type CinemaHandler struct {
	db *sql.DB
}

func NewCinemaHandler(db *sql.DB) *CinemaHandler {
	return &CinemaHandler{db: db}
}

// Create xử lý POST /cinemas.
func (h *CinemaHandler) Create(w http.ResponseWriter, r *http.Request) {
	var c models.Cinema
	if !decodeJSON(w, r, &c) {
		return
	}

	err := h.db.QueryRowContext(r.Context(), `
		INSERT INTO cinemas (name, address, city)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, c.Name, c.Address, c.City).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "tao cinema that bai: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, c)
}

// List xử lý GET /cinemas.
func (h *CinemaHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT id, name, address, city, created_at FROM cinemas ORDER BY created_at DESC
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "truy van danh sach cinema that bai: "+err.Error())
		return
	}
	defer rows.Close()

	cinemas := []models.Cinema{}
	for rows.Next() {
		var c models.Cinema
		if err := rows.Scan(&c.ID, &c.Name, &c.Address, &c.City, &c.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "doc du lieu cinema that bai: "+err.Error())
			return
		}
		cinemas = append(cinemas, c)
	}

	writeJSON(w, http.StatusOK, cinemas)
}

// GetByID xử lý GET /cinemas/{id}.
func (h *CinemaHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var c models.Cinema
	err := h.db.QueryRowContext(r.Context(), `
		SELECT id, name, address, city, created_at FROM cinemas WHERE id = $1
	`, id).Scan(&c.ID, &c.Name, &c.Address, &c.City, &c.CreatedAt)

	if isNoRows(err) {
		writeError(w, http.StatusNotFound, "khong tim thay cinema id="+id)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "truy van cinema that bai: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, c)
}
