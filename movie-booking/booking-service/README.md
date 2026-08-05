# booking-service

Chọn ghế, giữ ghế tạm thời (TTL qua Redis), confirm/cancel booking.

## Chạy local (docker-compose, ở thư mục cha `movie-booking/`)

```bash
docker compose up --build
```

## Test flow bằng curl

Giả sử đã có `movie_id`, `cinema_id`, `showtime_id` từ movie-service (xem README của movie-service).

```bash
# 1. Giữ ghế (hold) - mặc định TTL 300s (5 phút), chỉnh qua HOLD_TTL_SECONDS
curl -X POST http://localhost:8082/bookings/hold \
  -H "Content-Type: application/json" \
  -d '{"showtime_id":"<showtime_id>","user_id":"user-001","seats":["A1","A2"]}'

# 2. Xem sơ đồ ghế (ghế nào đang giữ / đã bán)
curl http://localhost:8082/showtimes/<showtime_id>/seats

# 3a. Confirm booking (giả lập payment-service gọi sau khi thanh toán thành công)
curl -X POST http://localhost:8082/bookings/<booking_id>/confirm

# 3b. Hoặc Cancel (nếu user đổi ý)
curl -X POST http://localhost:8082/bookings/<booking_id>/cancel

# 4. Xem chi tiết booking
curl http://localhost:8082/bookings/<booking_id>
```

## Test race condition (2 người cùng bấm 1 ghế)

Mở 2 terminal, chạy gần như đồng thời:

```bash
curl -X POST http://localhost:8082/bookings/hold -d '{"showtime_id":"X","user_id":"userA","seats":["A1"]}' &
curl -X POST http://localhost:8082/bookings/hold -d '{"showtime_id":"X","user_id":"userB","seats":["A1"]}' &
wait
```

Chỉ 1 trong 2 request nhận `201 Created`, request còn lại nhận `409 Conflict`
("ghế A1 đang được người khác giữ") — đây là hiệu ứng của `SETNX` trong Redis.

## Test TTL tự hết hạn

```bash
# Set HOLD_TTL_SECONDS=10 trong docker-compose.yml de test nhanh
curl -X POST http://localhost:8082/bookings/hold -d '{"showtime_id":"X","user_id":"userA","seats":["B5"]}'
sleep 12
curl http://localhost:8082/showtimes/X/seats   # B5 khong con trong "held" nua
```

## Chuẩn bị sẵn cho Phase 4 (k8s)

- `-wait-db -wait-redis -migrate` → command của **initContainer**
- mặc định → command của container **app**
- `-cleanup-expired-holds` → command của **CronJob** (Lab 1.3), nên chạy mỗi 1-5 phút
- `/healthz`, `/ready` → liveness/readiness (readiness check cả Postgres lẫn Redis)