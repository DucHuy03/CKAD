// Package config đọc cấu hình payment-service từ biến môi trường.
package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	// --- Postgres (luu lich su thanh toan) ---
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// --- RabbitMQ (publish event khi thanh toan thanh cong) ---
	RabbitMQURL string // vi du "amqp://guest:guest@rabbitmq:5672/"

	// --- Goi sang booking-service de confirm booking sau khi thanh toan thanh cong ---
	BookingServiceURL string

	HTTPPort int
}

func Load() Config {
	return Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvAsInt("DB_PORT", 5432),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "payment_service"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),

		BookingServiceURL: getEnv("BOOKING_SERVICE_URL", "http://localhost:8082"),

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
