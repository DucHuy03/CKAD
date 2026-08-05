# Dockerfile - multi-stage build cho booking-service (xem giải thích chi
# tiết từng bước ở movie-service/Dockerfile, ở đây dùng chung 1 pattern).

# ---------- Stage 1: builder ----------
FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/booking-service ./cmd/server/

# ---------- Stage 2: runtime ----------
FROM alpine:3.19

RUN apk add --no-cache ca-certificates
# Dung UID/GID co dinh de manifest Kubernetes co the cap quyen volume bang
# fsGroup=10001, tranh loi "permission denied" khi ghi file log.
RUN addgroup -g 10001 app && adduser -D -u 10001 -G app app
USER 10001

WORKDIR /app
COPY --from=builder /out/booking-service ./booking-service
COPY --from=builder /src/migrations ./migrations

EXPOSE 8080

# CMD mac dinh la mode server. initContainer se override bang:
#   ["-wait-db", "-wait-redis", "-migrate"]
# CronJob (Lab 1.3) se override bang:
#   ["-cleanup-expired-holds"]
ENTRYPOINT ["./booking-service"]
