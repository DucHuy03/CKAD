# payment-service

Xử lý thanh toán (giả lập gateway), confirm booking, publish event RabbitMQ.

## Chạy local (docker-compose, ở thư mục cha `movie-booking/`)

```bash
docker compose up --build
```

RabbitMQ Web UI (xem exchange/queue trực quan): http://localhost:15672 (guest/guest)

## Test flow bằng curl

Giả sử đã có `booking_id` từ bước `POST /bookings/hold` ở booking-service (status đang là HOLD).

```bash
# Thanh toán thành công (~90% xác suất giả lập ngẫu nhiên)
curl -X POST http://localhost:8083/payments/process \
  -H "Content-Type: application/json" \
  -d '{"booking_id":"<booking_id>","showtime_id":"<showtime_id>","user_id":"user-001","amount":240000,"method":"CARD","seats":["A1","A2"]}'

# Test có chủ đích nhánh THẤT BẠI (dùng method đặc biệt "FAIL_TEST")
curl -X POST http://localhost:8083/payments/process \
  -H "Content-Type: application/json" \
  -d '{"booking_id":"<booking_id>","showtime_id":"<showtime_id>","user_id":"user-001","amount":240000,"method":"FAIL_TEST","seats":["A1"]}'

# Xem chi tiết payment
curl http://localhost:8083/payments/<payment_id>
```

Sau khi thany toán SUCCESS, kiểm tra lại booking bên booking-service - status
phải chuyển từ `HOLD` sang `CONFIRMED`:

```bash
curl http://localhost:8082/bookings/<booking_id>
```

## Xem event trên RabbitMQ UI

1. Vào http://localhost:15672 → tab **Exchanges** → thấy `booking.events` (loại `topic`)
2. Vì chưa có consumer (notification-service chưa build), message publish
   với routing key `payment.success` sẽ **không có queue nào nhận** (bị
   "unroutable" và mất, vì không có `mandatory=true`) - đây là điều BÌNH
   THƯỜNG ở giai đoạn này. Khi làm `notification-service` (phase tiếp theo),
   nó sẽ tự khai báo queue + bind vào exchange này, từ đó mới nhận được message.

## Chuẩn bị sẵn cho Phase 4 (k8s)

- `-wait-db -wait-rabbitmq -migrate` → command của **initContainer**
- mặc định → command của container **app**
- `/healthz`, `/ready` → readiness check cả Postgres lẫn RabbitMQ channel