// Package qrcode tạo mã QR dưới dạng ảnh PNG (bytes), nhúng thông tin vé để
// nhân viên soát vé tại rạp có thể quét và xác nhận.
package qrcode

import (
	"encoding/base64"
	"fmt"

	"github.com/skip2/go-qrcode"
)

// TicketPayload là nội dung sẽ được mã hoá vào QR.
// Dùng format đơn giản "key=value;key=value" thay vì JSON đầy đủ để mã QR
// gọn hơn (QR code có giới hạn dung lượng, JSON có nhiều dấu ngoặc/ngoặc kép
// tốn byte hơn không cần thiết cho nhu cầu chỉ cần soát vé bằng mắt/máy quét đơn giản).
type TicketPayload struct {
	BookingID  string
	ShowtimeID string
	UserID     string
}

// GeneratePNG tạo ảnh QR (256x256px) chứa thông tin vé, trả về bytes PNG.
func GeneratePNG(payload TicketPayload) ([]byte, error) {
	content := fmt.Sprintf("booking_id=%s;showtime_id=%s;user_id=%s",
		payload.BookingID, payload.ShowtimeID, payload.UserID)

	png, err := qrcode.Encode(content, qrcode.Medium, 256)
	if err != nil {
		return nil, fmt.Errorf("tao ma QR that bai: %w", err)
	}
	return png, nil
}

// GeneratePNGBase64 giống GeneratePNG nhưng trả về chuỗi base64 - tiện để
// lưu thẳng vào cột TEXT trong Postgres (qr_data_base64) mà không cần thêm
// object storage (S3/MinIO) cho phạm vi bài lab này.
func GeneratePNGBase64(payload TicketPayload) (string, error) {
	png, err := GeneratePNG(payload)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(png), nil
}
