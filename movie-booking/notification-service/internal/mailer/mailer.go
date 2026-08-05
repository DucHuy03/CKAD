// Package mailer gửi email thật qua giao thức SMTP (chỉ dùng thư viện
// chuẩn net/smtp + mime/multipart, không cần thêm dependency ngoài).
//
// Khi chạy local (docker-compose), SMTP_HOST trỏ tới MailHog - 1 SMTP
// server giả lập chỉ dùng để test, không gửi email thật ra ngoài internet,
// có Web UI xem lại toàn bộ email đã "gửi" tại http://localhost:8025.
package mailer

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
)

type Config struct {
	Host string
	Port int
	From string
}

// SendTicketEmail dựng 1 email dạng MIME multipart gồm:
//   - Phần text/plain: nội dung thông báo đặt vé thành công
//   - Phần image/png (đính kèm, base64): ảnh QR vé
//
// rồi gửi qua SMTP tới địa chỉ "to".
func SendTicketEmail(cfg Config, to, subject, textBody string, qrPNG []byte) error {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// --- Header chung cua email ---
	buf.WriteString(fmt.Sprintf("From: %s\r\n", cfg.From))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", to))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n", writer.Boundary()))
	buf.WriteString("\r\n")

	// --- Phan 1: noi dung text ---
	textPart, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Type": {"text/plain; charset=UTF-8"},
	})
	if err != nil {
		return fmt.Errorf("tao phan text email that bai: %w", err)
	}
	if _, err := textPart.Write([]byte(textBody)); err != nil {
		return fmt.Errorf("ghi noi dung text email that bai: %w", err)
	}

	// --- Phan 2: anh QR dinh kem (encode base64 theo chuan MIME) ---
	imgPart, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"image/png"},
		"Content-Transfer-Encoding": {"base64"},
		"Content-Disposition":       {`attachment; filename="ticket-qr.png"`},
	})
	if err != nil {
		return fmt.Errorf("tao phan dinh kem QR that bai: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(qrPNG)
	if _, err := imgPart.Write([]byte(encoded)); err != nil {
		return fmt.Errorf("ghi du lieu QR that bai: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("dong multipart writer that bai: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	// MailHog (moi truong test) khong yeu cau authentication, nen auth = nil.
	// Voi SMTP server that co bat auth (SMTP_USER/SMTP_PASSWORD), can doi
	// sang smtp.PlainAuth(...) truyen vao SendMail.
	if err := smtp.SendMail(addr, nil, cfg.From, []string{to}, buf.Bytes()); err != nil {
		return fmt.Errorf("gui email qua SMTP %s that bai: %w", addr, err)
	}

	return nil
}
