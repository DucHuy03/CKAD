// Package config đọc cấu hình booking-service từ biến môi trường.
// So với movie-service, thêm 3 nhóm cấu hình mới: Redis, địa chỉ gọi sang
// movie-service, và thời gian giữ ghế (hold TTL).
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	// --- Postgres (luu tru lich su booking) ---
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// --- Redis (giu ghe co TTL) ---
	RedisAddr     string // vi du "redis:6379"
	RedisPassword string
	RedisDB       int

	// --- Goi sang movie-service de lay thong tin showtime ---
	MovieServiceURL string // vi du "http://movie-service:8080"

	// --- Nghiep vu ---
	HoldTTL time.Duration // thoi gian giu ghe truoc khi tu dong het han

	HTTPPort int
}

func Load() Config {
	return Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvAsInt("DB_PORT", 5432),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "booking_service"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),

		MovieServiceURL: getEnv("MOVIE_SERVICE_URL", "http://localhost:8081"),

		HoldTTL: time.Duration(getEnvAsInt("HOLD_TTL_SECONDS", 300)) * time.Second, // mac dinh 5 phut

		HTTPPort: getEnvAsInt("HTTP_PORT", 8080),
	}
}

func (c Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
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
