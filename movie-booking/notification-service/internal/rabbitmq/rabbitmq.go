// Package rabbitmq phía notification-service làm việc NGƯỢC LẠI với
// payment-service: thay vì publish, ở đây ta khai báo 1 QUEUE riêng và
// BIND nó vào exchange "booking.events" (đã được payment-service tạo sẵn)
// với routing key "payment.success" - từ lúc bind xong, mọi message
// payment-service publish với routing key đó sẽ được RabbitMQ tự động copy
// vào queue này để notification-service tiêu thụ.
package rabbitmq

import (
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeName = "booking.events"
	QueueName    = "notification.email" // ten queue rieng cua notification-service
	RoutingKey   = "payment.success"
)

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

// SetupConsumerTopology đảm bảo: exchange tồn tại (idempotent, dù
// payment-service thường tạo trước) -> queue riêng tồn tại -> queue được
// bind vào exchange với đúng routing key. Gọi lại nhiều lần vẫn an toàn.
//
// durable=true cho queue: nếu notification-service down, message vẫn nằm
// chờ trong queue (không mất), consumer online lại sẽ tiếp tục xử lý -
// đây là điểm khác biệt quan trọng so với việc "chờ ai đó publish lại".
func SetupConsumerTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(ExchangeName, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("khai bao exchange that bai: %w", err)
	}

	_, err := ch.QueueDeclare(
		QueueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("khai bao queue that bai: %w", err)
	}

	if err := ch.QueueBind(QueueName, RoutingKey, ExchangeName, false, nil); err != nil {
		return fmt.Errorf("bind queue vao exchange that bai: %w", err)
	}

	return nil
}

// Consume trả về 1 channel Go nhận message từ queue - dùng
// "manual ack" (autoAck=false) để chỉ xác nhận đã xử lý xong SAU KHI
// consumer.go xử lý thành công (ghi DB + gửi email), tránh mất message
// nếu notification-service crash giữa chừng lúc đang xử lý.
func Consume(ch *amqp.Channel) (<-chan amqp.Delivery, error) {
	// Prefetch = 1: chi nhan 1 message moi tai 1 thoi diem cho toi khi ack
	// message truoc do - tranh 1 consumer bi "ngop" neu co qua nhieu message
	// don don den cung luc trong khi xu ly (gui email) cham hon toc do nhan.
	if err := ch.Qos(1, 0, false); err != nil {
		return nil, fmt.Errorf("set QoS that bai: %w", err)
	}

	return ch.Consume(
		QueueName,
		"",    // consumer tag, de rong cho RabbitMQ tu sinh
		false, // autoAck = false -> manual ack
		false, // exclusive
		false, // no-local (khong ho tro tren RabbitMQ, luon false)
		false, // no-wait
		nil,
	)
}
