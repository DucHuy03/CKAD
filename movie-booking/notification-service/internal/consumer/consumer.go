// Package consumer chứa vòng lặp xử lý message chạy NỀN (1 goroutine riêng,
// song song với HTTP server) - đây là điểm khác biệt kiến trúc lớn nhất của
// notification-service so với 3 service trước (vốn chỉ phản ứng theo request).
package consumer

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"strconv"

	amqp "github.com/rabbitmq/amqp091-go"

	"notification-service/internal/mailer"
	"notification-service/internal/models"
	"notification-service/internal/qrcode"
	"notification-service/internal/rabbitmq"
)

type Consumer struct {
	db        *sql.DB
	mailerCfg mailer.Config
}

func New(db *sql.DB, mailerCfg mailer.Config) *Consumer {
	return &Consumer{db: db, mailerCfg: mailerCfg}
}

// Run bắt đầu vòng lặp nhận message - HÀM NÀY BLOCK MÃI MÃI, nên phải gọi
// bằng "go consumer.Run(ch)" từ main.go để không chặn HTTP server khởi động.
//
// FLOW xử lý MỖI message:
//  1. Parse JSON body thành PaymentSuccessEvent
//  2. Tạo mã QR (qrcode package)
//  3. Gửi email kèm QR (mailer package)
//  4. Ghi kết quả vào Postgres, dùng INSERT ... ON CONFLICT DO NOTHING theo
//     payment_id để IDEMPOTENT: nếu RabbitMQ redeliver message này lần 2
//     (do lần 1 consumer crash trước khi kịp ack), ta không gửi trùng email.
//  5. Ack message CHỈ SAU KHI xử lý xong - nếu code panic/crash giữa chừng
//     TRƯỚC bước ack, RabbitMQ tự động coi message là "chưa xử lý" và giao
//     lại (redeliver) cho consumer khác (hoặc chính consumer này sau khi restart).
func (c *Consumer) Run(ch *amqp.Channel) {
	deliveries, err := rabbitmq.Consume(ch)
	if err != nil {
		log.Fatalf("khong the bat dau consume: %v", err)
	}

	log.Printf("consumer da san sang, dang lang nghe queue %q", rabbitmq.QueueName)

	for msg := range deliveries {
		c.handleMessage(msg)
	}

	// Vong lap "for range" chi thoat khi channel deliveries dong (vi du mat
	// ket noi RabbitMQ) - log ro de biet consumer da ngung hoat dong, tranh
	// truong hop Pod van "chay" nhung khong con xu ly message nao.
	log.Println("canh bao: vong lap consumer da dung (co the do mat ket noi rabbitmq)")
}

func (c *Consumer) handleMessage(msg amqp.Delivery) {
	ctx := context.Background()

	var event models.PaymentSuccessEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("loi parse message (bo qua, ack de khong ket vao queue mai): %v", err)
		msg.Ack(false) // message loi dinh dang khong bao gio parse duoc du co redeliver bao nhieu lan - ack de bo qua
		return
	}

	log.Printf("dang xu ly payment_id=%s booking_id=%s", event.PaymentID, event.BookingID)

	// Email chi la gia lap (chua co user-service quan ly email thuc), dung
	// user_id lam dia chi de co the test duoc voi MailHog.
	email := event.UserID + "@example.com"

	payload := qrcode.TicketPayload{
		BookingID:  event.BookingID,
		ShowtimeID: event.ShowtimeID,
		UserID:     event.UserID,
	}

	qrPNG, qrErr := qrcode.GeneratePNG(payload)
	status := models.StatusSent
	failureReason := ""

	if qrErr != nil {
		status = models.StatusFailed
		failureReason = "tao QR that bai: " + qrErr.Error()
	} else {
		subject := "Ve xem phim cua ban da duoc xac nhan"
		amountStr := strconv.FormatFloat(event.Amount, 'f', 0, 64)
		body := "Cam on ban da dat ve! Booking ID: " + event.BookingID +
			"\nSuat chieu: " + event.ShowtimeID +
			"\nSo tien da thanh toan: " + amountStr + " VND" +
			"\nVui long xuat trinh ma QR dinh kem khi vao rap."

		if sendErr := mailer.SendTicketEmail(c.mailerCfg, email, subject, body, qrPNG); sendErr != nil {
			status = models.StatusFailed
			failureReason = "gui email that bai: " + sendErr.Error()
		}
	}

	qrBase64 := ""
	if qrErr == nil {
		if b64, encErr := qrcode.GeneratePNGBase64(payload); encErr == nil {
			qrBase64 = b64
		}
	}

	_, dbErr := c.db.ExecContext(ctx, `
		INSERT INTO notifications (payment_id, booking_id, showtime_id, user_id, email, status, failure_reason, qr_data_base64)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (payment_id) DO NOTHING
	`, event.PaymentID, event.BookingID, event.ShowtimeID, event.UserID, email, status, failureReason, qrBase64)

	if dbErr != nil {
		// Ghi DB that bai la tinh huong nghiem trong: khong biet chac message
		// nay da duoc xu ly hay chua -> KHONG ack, de RabbitMQ redeliver,
		// tha con hon bo sot 1 email ve.
		log.Printf("loi ghi notification vao DB (se KHONG ack, cho redeliver): %v", dbErr)
		msg.Nack(false, true) // requeue = true
		return
	}

	if status == models.StatusFailed {
		log.Printf("xu ly notification cho payment_id=%s that bai: %s", event.PaymentID, failureReason)
	} else {
		log.Printf("da gui email + QR thanh cong cho payment_id=%s toi %s", event.PaymentID, email)
	}

	msg.Ack(false)
}
