# notification-service

Consume event `payment.success` từ RabbitMQ, tạo mã QR vé, gửi email xác nhận.

## Chạy local (docker-compose, ở thư mục cha `movie-booking/`)

```bash
docker compose up --build
```

MailHog Web UI (xem email đã "gửi"): http://localhost:8025
RabbitMQ Web UI: http://localhost:15672 (guest/guest)

## Test full flow end-to-end

Chạy tuần tự qua 4 service (đã build đủ cả 4):

```bash
# 1. Tạo cinema + movie + showtime (xem README movie-service)
# 2. Hold ghế (booking-service) -> lấy booking_id
# 3. Xử lý thanh toán (payment-service) -> publish event payment.success
curl -X POST http://localhost:8083/payments/process \
  -H "Content-Type: application/json" \
  -d '{"booking_id":"<booking_id>","showtime_id":"<showtime_id>","user_id":"user-001","amount":240000,"method":"CARD","seats":["A1","A2"]}'

# 4. Kiểm tra notification-service đã xử lý (đợi vài giây cho consumer chạy)
curl http://localhost:8084/notifications/by-booking/<booking_id>
```

Sau đó mở http://localhost:8025 (MailHog) - phải thấy 1 email mới với:
- Subject "Ve xem phim cua ban da duoc xac nhan"
- Nội dung text kèm booking_id, số tiền
- File đính kèm `ticket-qr.png` (mã QR)

## Xem log consumer

```bash
docker compose logs -f notification-service
```

Sẽ thấy dòng `consumer da san sang, dang lang nghe queue "notification.email"`
ngay khi service start, và log xử lý từng message khi có payment thành công.

## Test idempotency (không gửi trùng email)

Publish lại cùng 1 `payment_id` (ví dụ restart notification-service khi
RabbitMQ chưa kịp xoá message do consumer chưa ack) sẽ KHÔNG tạo thêm dòng
notification mới nhờ `UNIQUE INDEX` + `ON CONFLICT DO NOTHING` trên
`payment_id` — kiểm tra bằng cách query trực tiếp Postgres:

```bash
docker compose exec postgres-notification psql -U postgres -d notification_service \
  -c "SELECT payment_id, count(*) FROM notifications GROUP BY payment_id;"
```

Mọi `count` phải bằng 1.

## Chuẩn bị sẵn cho Phase 4 (k8s)

- `-wait-db -wait-rabbitmq -migrate` → command của **initContainer**
- mặc định → command của container **app** (chạy cả HTTP server lẫn consumer goroutine)
- `/healthz`, `/ready` → liveness/readiness
- Điểm cần lưu ý khi viết Deployment sau này: **không nên scale
  notification-service lên nhiều replica một cách vô tư** nếu chưa hiểu rõ
  RabbitMQ tự chia message cho các consumer cùng queue như thế nào (round-robin
  theo prefetch) - nhưng nhờ thiết kế idempotent (UNIQUE payment_id) ở đây,
  dù có 2 Pod cùng nhận trùng 1 message thì cũng không gửi email trùng.