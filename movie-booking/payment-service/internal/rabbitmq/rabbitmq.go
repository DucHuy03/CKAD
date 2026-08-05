// Package rabbitmq quản lý kết nối RabbitMQ và việc publish event.
//
// THIẾT KẾ EXCHANGE/ROUTING KEY:
// Dùng 1 "topic exchange" tên "booking.events" thay vì publish thẳng vào 1
// queue cụ thể. Lý do: payment-service KHÔNG cần biết ai đang lắng nghe event
// của nó (notification-service, hay sau này thêm 1 service report/analytics
// nào đó) - payment-service chỉ publish với routing key "payment.success" và
// "quên đi", ai muốn nghe thì tự khai báo queue riêng và BIND vào exchange
// này với routing key tương ứng (notification-service sẽ làm việc đó ở
// service tiếp theo). Đây là pattern pub/sub kinh điển, tách rời publisher
// khỏi (các) consumer - thêm consumer mới không cần sửa payment-service.
package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const ExchangeName = "booking.events"

// Connect mở 1 connection + 1 channel tới RabbitMQ.
// Trong AMQP, "connection" là kết nối TCP, "channel" là 1 luồng logic ảo
// multiplex trên connection đó - hầu hết thao tác (publish, consume,
// declare exchange/queue) đều thực hiện qua channel, không phải connection.
func Connect(url string) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, nil, fmt.Errorf("ket noi rabbitmq that bai: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("mo channel rabbitmq that bai: %w", err)
	}

	return conn, ch, nil
}

// WaitForRabbitMQ liên tục thử Dial() cho tới khi thành công hoặc hết
// timeout - dùng trong initContainer, cùng lý do với WaitForDB/WaitForRedis:
// RabbitMQ Pod có thể chưa Ready khi payment-service Pod đã được schedule.
func WaitForRabbitMQ(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		conn, err := amqp.Dial(url)
		if err == nil {
			conn.Close()
			return nil
		}
		lastErr = err
		log.Printf("cho rabbitmq san sang... (%v)", lastErr)
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("het thoi gian cho rabbitmq sau %s: %w", timeout, lastErr)
}

// DeclareExchange đảm bảo exchange "booking.events" tồn tại trước khi publish.
// ExchangeDeclare là idempotent (gọi nhiều lần với cùng tham số không lỗi),
// nên gọi lại mỗi lần service start là an toàn.
//
// durable=true: exchange được RabbitMQ lưu xuống đĩa, sống sót qua broker
// restart - phù hợp với event quan trọng như "thanh toán thành công",
// không muốn mất nếu RabbitMQ Pod bị restart giữa chừng.
func DeclareExchange(ch *amqp.Channel) error {
	return ch.ExchangeDeclare(
		ExchangeName,
		"topic", // loai exchange: routing key co the dung wildcard (vd "payment.*")
		true,    // durable
		false,   // auto-deleted
		false,   // internal
		false,   // no-wait
		nil,     // arguments
	)
}

// PaymentSuccessEvent là payload publish lên RabbitMQ khi thanh toán thành công.
// notification-service (Phase tiếp theo) sẽ deserialize đúng struct này.
type PaymentSuccessEvent struct {
	PaymentID  string   `json:"payment_id"`
	BookingID  string   `json:"booking_id"`
	ShowtimeID string   `json:"showtime_id"`
	UserID     string   `json:"user_id"`
	Amount     float64  `json:"amount"`
	Seats      []string `json:"seats"`
	PaidAt     string   `json:"paid_at"` // RFC3339, dung string cho don gian khi serialize/deserialize giua service
}

// PublishPaymentSuccess publish event "payment.success" lên exchange.
func PublishPaymentSuccess(ctx context.Context, ch *amqp.Channel, event PaymentSuccessEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal payment success event that bai: %w", err)
	}

	return ch.PublishWithContext(ctx,
		ExchangeName,
		"payment.success", // routing key
		false,             // mandatory
		false,             // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // message duoc RabbitMQ ghi xuong dia, khong mat khi broker restart
			Body:         body,
		},
	)
}
