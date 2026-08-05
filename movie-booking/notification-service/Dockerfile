# Dockerfile - multi-stage build cho notification-service.

FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/notification-service ./cmd/server/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
# UID/GID co dinh khop voi fsGroup trong Deployment, de app ghi duoc log.
RUN addgroup -g 10001 app && adduser -D -u 10001 -G app app
USER 10001

WORKDIR /app
COPY --from=builder /out/notification-service ./notification-service
COPY --from=builder /src/migrations ./migrations

EXPOSE 8080

# CMD mac dinh la mode server (chay ca HTTP server lan consumer goroutine).
# initContainer se override bang: ["-wait-db", "-wait-rabbitmq", "-migrate"]
ENTRYPOINT ["./notification-service"]
