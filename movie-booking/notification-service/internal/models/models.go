package models

import "time"

type NotificationStatus string

const (
	StatusPending NotificationStatus = "PENDING"
	StatusSent    NotificationStatus = "SENT"
	StatusFailed  NotificationStatus = "FAILED"
)

// Notification ánh xạ 1 dòng trong bảng notifications.
type Notification struct {
	ID            string             `json:"id"`
	PaymentID     string             `json:"payment_id"`
	BookingID     string             `json:"booking_id"`
	ShowtimeID    string             `json:"showtime_id"`
	UserID        string             `json:"user_id"`
	Email         string             `json:"email"`
	Status        NotificationStatus `json:"status"`
	FailureReason string             `json:"failure_reason,omitempty"`
	QRDataBase64  string             `json:"qr_data_base64,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

// PaymentSuccessEvent PHẢI khớp hoàn toàn với struct cùng tên bên
// payment-service (internal/rabbitmq/rabbitmq.go) - đây là "hợp đồng ngầm"
// giữa 2 service qua message queue. Nếu 1 bên đổi field mà bên kia không
// đổi theo, lỗi sẽ chỉ lộ ra lúc runtime (parse JSON thiếu field), không có
// compiler nào bắt được lỗi này - đây là nhược điểm cố hữu của giao tiếp
// qua message queue so với gọi hàm trực tiếp, cần lưu ý khi mở rộng hệ thống.
type PaymentSuccessEvent struct {
	PaymentID  string   `json:"payment_id"`
	BookingID  string   `json:"booking_id"`
	ShowtimeID string   `json:"showtime_id"`
	UserID     string   `json:"user_id"`
	Amount     float64  `json:"amount"`
	Seats      []string `json:"seats"`
	PaidAt     string   `json:"paid_at"`
}
