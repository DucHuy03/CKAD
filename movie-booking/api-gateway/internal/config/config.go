// Package config đọc cấu hình api-gateway từ biến môi trường.
//
// Khác với 4 service trước, api-gateway không cần cấu hình DB - nó không
// lưu trạng thái gì cả (stateless), chỉ cần biết: URL của 4 backend để
// route request tới, secret để ký/xác minh JWT, và ngưỡng rate limit.
package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	// --- URL cac backend de proxy request toi ---
	// Hom nay (docker-compose) tro thang toi service, sau nay (Phase 4 k8s)
	// se doi thanh "http://localhost:<port>" tro toi ambassador nginx cung Pod.
	MovieServiceURL        string
	BookingServiceURL      string
	PaymentServiceURL      string
	NotificationServiceURL string

	// --- JWT ---
	JWTSecret     string
	JWTExpiryMins int

	// --- Rate limit ---
	RateLimitPerMinute int

	HTTPPort int
}

func Load() Config {
	return Config{
		MovieServiceURL:        getEnv("MOVIE_SERVICE_URL", "http://localhost:8081"),
		BookingServiceURL:      getEnv("BOOKING_SERVICE_URL", "http://localhost:8082"),
		PaymentServiceURL:      getEnv("PAYMENT_SERVICE_URL", "http://localhost:8083"),
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8084"),

		// CANH BAO: gia tri default nay CHI danh cho dev local. Trong k8s,
		// JWT_SECRET PHAI duoc bom qua Secret (Lab 3.1), khong bao gio dung
		// gia tri mac dinh nay trong moi truong that.
		JWTSecret:     getEnv("JWT_SECRET", "dev-only-insecure-secret-please-override"),
		JWTExpiryMins: getEnvAsInt("JWT_EXPIRY_MINUTES", 60),

		RateLimitPerMinute: getEnvAsInt("RATE_LIMIT_PER_MINUTE", 60),

		HTTPPort: getEnvAsInt("HTTP_PORT", 8080),
	}
}

func (c Config) JWTExpiryDuration() time.Duration {
	return time.Duration(c.JWTExpiryMins) * time.Minute
}

func getEnv(key, defaultVal string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return defaultVal
}

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
