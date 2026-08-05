package handlers

import (
	"database/sql"
	"net/http"

	"notification-service/internal/models"
)

// NotificationHandler chỉ có 1 chức năng đọc: tra cứu lại xem 1 booking đã
// được gửi email/QR chưa - hữu ích để debug ("tại sao user báo chưa nhận
// được vé?") mà không cần đọc trực tiếp log của Pod.
type NotificationHandler struct {
	db *sql.DB
}

func NewNotificationHandler(db *sql.DB) *NotificationHandler {
	return &NotificationHandler{db: db}
}

// GetByBookingID xử lý GET /notifications/by-booking/{booking_id}.
func (h *NotificationHandler) GetByBookingID(w http.ResponseWriter, r *http.Request) {
	bookingID := r.PathValue("booking_id")

	var n models.Notification
	err := h.db.QueryRowContext(r.Context(), `
		SELECT id, payment_id, booking_id, showtime_id, user_id, email, status, failure_reason, qr_data_base64, created_at, updated_at
		FROM notifications WHERE booking_id = $1
		ORDER BY created_at DESC LIMIT 1
	`, bookingID).Scan(&n.ID, &n.PaymentID, &n.BookingID, &n.ShowtimeID, &n.UserID,
		&n.Email, &n.Status, &n.FailureReason, &n.QRDataBase64, &n.CreatedAt, &n.UpdatedAt)

	if isNoRows(err) {
		writeError(w, http.StatusNotFound, "chua co notification nao cho booking_id="+bookingID)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "truy van notification that bai: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, n)
}
