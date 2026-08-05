package handlers

import (
	"database/sql"
	"net/http"

	"movie-service/internal/models"
)

// MovieHandler gom các HTTP handler thao tác trên resource "movie".
// Giữ *sql.DB trực tiếp (không qua ORM) để dev quen SQL thuần dễ đọc,
// đồng thời tránh thêm dependency ngoài không cần thiết.
type MovieHandler struct {
	db *sql.DB
}

// NewMovieHandler khởi tạo MovieHandler với connection pool đã có sẵn.
func NewMovieHandler(db *sql.DB) *MovieHandler {
	return &MovieHandler{db: db}
}

// Create xử lý POST /movies.
// Flow: decode JSON body -> INSERT -> đọc lại các cột do DB tự sinh
// (id, created_at, updated_at) bằng mệnh đề RETURNING -> trả về 201.
func (h *MovieHandler) Create(w http.ResponseWriter, r *http.Request) {
	var m models.Movie
	if !decodeJSON(w, r, &m) {
		return // decodeJSON da tu ghi loi 400 vao response roi
	}

	query := `
		INSERT INTO movies (title, description, duration_minutes, genre, release_date, poster_url)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	err := h.db.QueryRowContext(r.Context(), query,
		m.Title, m.Description, m.DurationMinutes, m.Genre, m.ReleaseDate, m.PosterURL,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "tao movie that bai: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, m)
}

// List xử lý GET /movies - trả về toàn bộ danh sách phim.
// Với hệ thống thật nên thêm phân trang (LIMIT/OFFSET hoặc cursor), ở đây
// bỏ qua để giữ ví dụ gọn cho mục đích học k8s.
func (h *MovieHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT id, title, description, duration_minutes, genre, release_date, poster_url, created_at, updated_at
		FROM movies
		ORDER BY created_at DESC
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "truy van danh sach movie that bai: "+err.Error())
		return
	}
	defer rows.Close()

	movies := []models.Movie{} // khoi tao slice rong (khong phai nil) de tra ve "[]" thay vi "null" khi khong co du lieu
	for rows.Next() {
		var m models.Movie
		if err := rows.Scan(&m.ID, &m.Title, &m.Description, &m.DurationMinutes,
			&m.Genre, &m.ReleaseDate, &m.PosterURL, &m.CreatedAt, &m.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "doc du lieu movie that bai: "+err.Error())
			return
		}
		movies = append(movies, m)
	}

	writeJSON(w, http.StatusOK, movies)
}

// GetByID xử lý GET /movies/{id}.
func (h *MovieHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id") // Go 1.22+ ho tro lay path param truc tiep tu ServeMux, khong can router ngoai

	var m models.Movie
	err := h.db.QueryRowContext(r.Context(), `
		SELECT id, title, description, duration_minutes, genre, release_date, poster_url, created_at, updated_at
		FROM movies WHERE id = $1
	`, id).Scan(&m.ID, &m.Title, &m.Description, &m.DurationMinutes,
		&m.Genre, &m.ReleaseDate, &m.PosterURL, &m.CreatedAt, &m.UpdatedAt)

	if isNoRows(err) {
		writeError(w, http.StatusNotFound, "khong tim thay movie id="+id)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "truy van movie that bai: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, m)
}

// Update xử lý PUT /movies/{id} - cập nhật toàn bộ field (không phải PATCH từng phần).
func (h *MovieHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var m models.Movie
	if !decodeJSON(w, r, &m) {
		return
	}

	query := `
		UPDATE movies
		SET title = $1, description = $2, duration_minutes = $3,
		    genre = $4, release_date = $5, poster_url = $6, updated_at = now()
		WHERE id = $7
		RETURNING id, created_at, updated_at
	`
	err := h.db.QueryRowContext(r.Context(), query,
		m.Title, m.Description, m.DurationMinutes, m.Genre, m.ReleaseDate, m.PosterURL, id,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)

	if isNoRows(err) {
		writeError(w, http.StatusNotFound, "khong tim thay movie id="+id)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cap nhat movie that bai: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, m)
}

// Delete xử lý DELETE /movies/{id}.
// Lưu ý: nếu movie này đang có showtime tham chiếu, ràng buộc ON DELETE CASCADE
// trong migration sẽ tự xoá luôn các showtime liên quan.
func (h *MovieHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	result, err := h.db.ExecContext(r.Context(), `DELETE FROM movies WHERE id = $1`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "xoa movie that bai: "+err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "khong tim thay movie id="+id)
		return
	}

	// 204 No Content: xoa thanh cong, khong can tra body.
	w.WriteHeader(http.StatusNoContent)
}
