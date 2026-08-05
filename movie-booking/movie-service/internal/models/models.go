// Package models định nghĩa các struct ánh xạ trực tiếp tới 3 bảng trong
// migrations/001_init.sql. Field nào NULL được trong DB thì dùng con trỏ
// (*string, *time.Time...) để phân biệt "rỗng" và "không có giá trị" - quen
// thuộc với ai đã quen làm việc với NULL trong C (con trỏ NULL = không có dữ liệu).
package models

import "time"

// Movie ánh xạ 1 dòng trong bảng movies.
type Movie struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	DurationMinutes int        `json:"duration_minutes"`
	Genre           string     `json:"genre"`
	ReleaseDate     *time.Time `json:"release_date,omitempty"` // co the NULL trong DB
	PosterURL       string     `json:"poster_url"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Cinema ánh xạ 1 dòng trong bảng cinemas.
type Cinema struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	City      string    `json:"city"`
	CreatedAt time.Time `json:"created_at"`
}

// Showtime ánh xạ 1 dòng trong bảng showtimes.
// booking-service se goi GET /showtimes/{id} va doc struct nay (qua JSON)
// de biet gia ve (Price) va tong so ghe (TotalSeats) truoc khi cho user chon ghe.
type Showtime struct {
	ID         string    `json:"id"`
	MovieID    string    `json:"movie_id"`
	CinemaID   string    `json:"cinema_id"`
	RoomName   string    `json:"room_name"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Price      float64   `json:"price"`
	TotalSeats int       `json:"total_seats"`
	CreatedAt  time.Time `json:"created_at"`
}
