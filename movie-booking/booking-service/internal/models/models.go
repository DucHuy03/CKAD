package models

import (
	"time"

	"github.com/lib/pq"
)

// Trạng thái của 1 booking - dùng type riêng (thay vì string trần) để
// compiler bắt lỗi gõ nhầm ngay lúc build, thay vì phải chờ tới runtime.
type BookingStatus string

const (
	StatusHold      BookingStatus = "HOLD"
	StatusConfirmed BookingStatus = "CONFIRMED"
	StatusCancelled BookingStatus = "CANCELLED"
	StatusExpired   BookingStatus = "EXPIRED"
)

// Booking ánh xạ 1 dòng trong bảng bookings.
//
// Seats dùng kiểu pq.StringArray (do driver lib/pq cung cấp) thay vì
// []string trần, vì database/sql không tự biết cách Scan() 1 cột kiểu
// TEXT[] của Postgres vào []string thường - pq.StringArray implement sẵn
// interface sql.Scanner/driver.Valuer để làm việc đó.
type Booking struct {
	ID         string         `json:"id"`
	ShowtimeID string         `json:"showtime_id"`
	UserID     string         `json:"user_id"`
	Seats      pq.StringArray `json:"seats"`
	TotalPrice float64        `json:"total_price"`
	Status     BookingStatus  `json:"status"`
	ExpiresAt  *time.Time     `json:"expires_at,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// HoldRequest là body của POST /bookings/hold.
type HoldRequest struct {
	ShowtimeID string   `json:"showtime_id"`
	UserID     string   `json:"user_id"`
	Seats      []string `json:"seats"`
}
