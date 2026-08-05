# api-gateway

Entry point duy nhất của hệ thống: routing, xác thực JWT, rate limit.

## Chạy local (docker-compose, ở thư mục cha `movie-booking/`)

```bash
docker compose up --build
```

Từ giờ **mọi request đi qua `api-gateway` ở port 8080**, không gọi thẳng
tới port riêng của từng service nữa (8081/8082/8083/8084 vẫn mở để debug
trực tiếp khi cần, nhưng luồng thật đi qua gateway).

## Test full flow qua gateway

```bash
# 1. Login lấy token (chấp nhận bất kỳ password nào - xem cảnh báo trong code)
TOKEN=$(curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user-001","password":"bất-kỳ"}' | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")

echo "Token: $TOKEN"

# 2. Xem phim - KHÔNG cần token (route công khai)
curl http://localhost:8080/api/movies

# 3. Tạo phim - CẦN token (route quản trị)
curl -X POST http://localhost:8080/api/movies \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Dune: Part Two","description":"Sci-fi","duration_minutes":166,"genre":"Sci-Fi"}'

# 4. Thử KHÔNG kèm token -> phải nhận 401
curl -i -X POST http://localhost:8080/api/movies \
  -H "Content-Type: application/json" \
  -d '{"title":"Test"}'

# 5. Hold ghế - CẦN token
curl -X POST http://localhost:8080/api/bookings/hold \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"showtime_id":"<showtime_id>","user_id":"user-001","seats":["A1"]}'
```

## Test rate limit

```bash
# RATE_LIMIT_PER_MINUTE mặc định 60 -> hạ xuống thấp để test nhanh, sửa trong
# docker-compose.yml thành "RATE_LIMIT_PER_MINUTE": "5", restart api-gateway,
# roi ban 10 request lien tuc:
for i in $(seq 1 10); do curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/api/movies; done
```

Sau khi vượt ngưỡng, các request tiếp theo trong cùng phút phải trả `429`.

## Kiểm tra thứ tự khởi động (init container mô phỏng)

```bash
docker compose logs api-gateway-init
```

Phải thấy log kiểu `cho movie-service san sang...` rồi `movie-service da san
sang` cho cả 4 backend, trước khi `api-gateway` (container chính) start.

## Chuẩn bị sẵn cho Phase 4 (k8s)

- `-wait-backends` → command của **initContainer** (khác các service trước:
  chờ SERVICE KHÁC thay vì DB/Redis/RabbitMQ)
- mặc định → command của container **app**
- `/healthz`, `/ready` → không phụ thuộc backend (xem lý do trong code)
- `JWT_SECRET` hiện đang đọc từ biến môi trường thường - ở Phase 4 (Lab 3.1)
  sẽ chuyển sang đọc từ **Secret**, không phải ConfigMap