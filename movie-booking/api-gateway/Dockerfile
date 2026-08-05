# Dockerfile - multi-stage build cho api-gateway.
# Khac cac service truoc: khong co thu muc migrations/ de copy, vi
# api-gateway khong co DB rieng.

FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api-gateway ./cmd/server/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
# UID/GID co dinh khop voi fsGroup trong Deployment, de app ghi duoc log.
RUN addgroup -g 10001 app && adduser -D -u 10001 -G app app
USER 10001

WORKDIR /app
COPY --from=builder /out/api-gateway ./api-gateway

EXPOSE 8080

# CMD mac dinh la mode server. initContainer se override bang: ["-wait-backends"]
ENTRYPOINT ["./api-gateway"]
