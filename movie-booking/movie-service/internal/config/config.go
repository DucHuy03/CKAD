// Package config chịu trách nhiệm đọc toàn bộ cấu hình của service từ biến môi trường.
//
// Lý do tách riêng package này: trong k8s, config sẽ được bơm vào Pod qua
// ConfigMap (non-sensitive: DB_HOST, DB_PORT, HTTP_PORT...) và Secret
// (sensitive: DB_PASSWORD...) dưới dạng biến môi trường (Lab 3.1). Code chỉ
// cần biết đọc env var, không cần biết giá trị đến từ ConfigMap hay Secret.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config gom toàn bộ giá trị cấu hình cần thiết để service chạy được.
type Config struct {
	// --- Cấu hình kết nối Postgres ---
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string // "disable" khi chạy local/docker-compose, "require" khi có TLS thật

	// --- Cấu hình HTTP server ---
	HTTPPort int

	// --- Cấu hình logging ---
	// Neu rong: chi ghi log ra stdout (mac dinh khi chay local khong co sidecar).
	// Neu co gia tri (vd "/var/log/app/app.log"): ghi ca stdout lan file do,
	// de sidecar log-shipper (Phase 2) tail duoc qua volume emptyDir dung chung.
	LogFilePath string
}

// Load đọc toàn bộ biến môi trường và trả về Config đã điền sẵn giá trị mặc định
// hợp lý cho môi trường local nếu biến môi trường không được set.
//
// Thiết kế "có default" giúp dev chạy `go run` ngay mà không cần export
// hàng loạt biến môi trường trước — nhưng khi deploy thật (docker-compose,
// k8s) thì luôn override đầy đủ qua ConfigMap/Secret, không dựa vào default.
func Load() Config {
	return Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvAsInt("DB_PORT", 5432),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "movie_service"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
		HTTPPort:   getEnvAsInt("HTTP_PORT", 8080),

		LogFilePath: getEnv("LOG_FILE_PATH", ""),
	}
}

// DSN dựng chuỗi kết nối Postgres (Data Source Name) theo format mà
// driver lib/pq hiểu, từ các trường rời rạc trong Config.
func (c Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

// getEnv đọc 1 biến môi trường dạng string, trả về defaultVal nếu chưa set.
func getEnv(key, defaultVal string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return defaultVal
}

// getEnvAsInt đọc 1 biến môi trường và parse sang int.
// Nếu biến không tồn tại hoặc parse lỗi (ví dụ user set nhầm giá trị không phải số),
// fallback về defaultVal thay vì crash ngay — lỗi thật sự (nếu có) sẽ lộ ra
// khi service cố dùng giá trị sai đó để connect DB, dễ debug hơn là panic ở bước đọc config.
func getEnvAsInt(key string, defaultVal int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}
