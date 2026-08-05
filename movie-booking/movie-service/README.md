# movie-service

CRUD service cho phim / rạp / suất chiếu, viết bằng Go + Postgres (database/sql + lib/pq).

## Chạy local (docker-compose)

```bash
cd movie-booking
docker compose up --build
```

Thứ tự compose sẽ tự thực hiện đúng như trong k8s sau này:
1. `postgres` start, chờ tới khi healthy (`pg_isready`)
2. `movie-service-migrate` chạy (`-wait-db -migrate`) → tạo bảng → exit 0
3. `movie-service` mới start, lắng nghe ở `localhost:8081`

## Test nhanh bằng curl

```bash
# Health check
curl http://localhost:8081/healthz
curl http://localhost:8081/ready

# Tạo 1 cinema
curl -X POST http://localhost:8081/cinemas \
  -H "Content-Type: application/json" \
  -d '{"name":"CGV Vincom","address":"72 Le Thanh Ton","city":"Ha Noi"}'

# Tạo 1 movie
curl -X POST http://localhost:8081/movies \
  -H "Content-Type: application/json" \
  -d '{"title":"Dune: Part Two","description":"Sci-fi epic","duration_minutes":166,"genre":"Sci-Fi","poster_url":""}'

# Lấy id vừa tạo (từ response ở trên) để tạo showtime, ví dụ:
curl -X POST http://localhost:8081/showtimes \
  -H "Content-Type: application/json" \
  -d '{"movie_id":"<movie_id>","cinema_id":"<cinema_id>","room_name":"Room 1","start_time":"2026-08-01T19:00:00Z","end_time":"2026-08-01T21:46:00Z","price":120000,"total_seats":80}'

# List
curl http://localhost:8081/movies
curl http://localhost:8081/showtimes?movie_id=<movie_id>
```

## Cấu trúc thư mục

```
movie-service/
  cmd/server/main.go          # entry point, 2 mode: server / wait-db+migrate
  internal/config/            # đọc env var
  internal/db/                # connect, wait-for-db, run migration
  internal/models/            # struct Movie/Cinema/Showtime
  internal/handlers/          # HTTP handler (CRUD)
  migrations/001_init.sql     # schema
  Dockerfile                  # multi-stage build
```

## Chuẩn bị sẵn cho Phase 4 (k8s)

- `-wait-db -migrate` → dùng làm command của **initContainer**
- mặc định (không flag) → dùng làm command của container **app**
- `/healthz`, `/ready` → dùng cho livenessProbe / readinessProbe (Lab 5.1)
- Chạy bằng user non-root trong Dockerfile → chuẩn bị cho Lab 3.2