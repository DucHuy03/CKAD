package models

import "time"

type PaymentStatus string

const (
	StatusPending PaymentStatus = "PENDING"
	StatusSuccess PaymentStatus = "SUCCESS"
	StatusFailed  PaymentStatus = "FAILED"
)

// Payment ánh xạ 1 dòng trong bảng payments.
type Payment struct {
	ID            string        `json:"id"`
	BookingID     string        `json:"booking_id"`
	ShowtimeID    string        `json:"showtime_id"`
	UserID        string        `json:"user_id"`
	Amount        float64       `json:"amount"`
	Method        string        `json:"method"`
	Status        PaymentStatus `json:"status"`
	FailureReason string        `json:"failure_reason,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

// ProcessRequest là body của POST /payments/process.
// Seats được truyền kèm (không tra lại DB) để publish event cho
// notification-service dùng luôn, đỡ phải notification-service gọi ngược
// lại booking-service lần nữa chỉ để lấy danh sách ghế.
type ProcessRequest struct {
	BookingID  string   `json:"booking_id"`
	ShowtimeID string   `json:"showtime_id"`
	UserID     string   `json:"user_id"`
	Amount     float64  `json:"amount"`
	Method     string   `json:"method"`
	Seats      []string `json:"seats"`
}
