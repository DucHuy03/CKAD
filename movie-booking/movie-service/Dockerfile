# Dockerfile - multi-stage build cho movie-service
#
# Stage 1 (builder): dùng image Go đầy đủ để compile ra 1 binary tĩnh.
# Stage 2 (runtime): copy binary + migrations/ sang image alpine tối giản,
# không mang theo toolchain Go, source code hay build cache -> image nhỏ,
# giảm bề mặt tấn công (đúng tinh thần chuẩn bị cho Lab 3.2 Security Context).

# ---------- Stage 1: builder ----------
FROM golang:1.24-alpine AS builder

WORKDIR /src

# Copy go.mod/go.sum truoc de tan dung Docker layer cache: neu chi sua code
# ma khong doi dependency, buoc "go mod download" khong phai chay lai.
COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# CGO_ENABLED=0: build binary tinh (khong phu thuoc libc cua he thong),
# chay duoc tren base image toi gian nhu alpine hay distroless/scratch.
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/movie-service ./cmd/server/

# ---------- Stage 2: runtime ----------
FROM alpine:3.19

# Cai ca-certificates de HTTPS request (neu sau nay goi service khac qua TLS) hoat dong.
RUN apk add --no-cache ca-certificates

# Tao user khong phai root - chuan bi san cho Lab 3.2 (runAsNonRoot).
# RUN addgroup -S app && adduser -S app -G app
# USER app
RUN addgroup -g 10001 app \
 && adduser -D -u 10001 -G app app

USER 10001
WORKDIR /app
COPY --from=builder /out/movie-service ./movie-service
COPY --from=builder /src/migrations ./migrations

EXPOSE 8080

USER 10001
# ENTRYPOINT co dinh, CMD la default args - k8s se override CMD bang args
# khac nhau giua initContainer (["-wait-db", "-migrate"]) va app container
# (khong truyen gi, dung mode server mac dinh). Xem Phase 4.
ENTRYPOINT ["./movie-service"]