package middleware

import (
	"net/http"
	"strings"

	"api-gateway/internal/ratelimit"
)

// RateLimit trả về 1 middleware giới hạn số request mỗi client IP.
func RateLimit(limiter *ratelimit.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := clientKey(r)

			if !limiter.Allow(key) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"vuot qua gioi han so luong request, thu lai sau"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// clientKey xác định "danh tính" của client để áp rate limit theo từng
// client riêng biệt (không phải giới hạn chung cho toàn bộ traffic).
//
// Ưu tiên header X-Forwarded-For nếu có: khi sau này api-gateway đứng sau
// 1 load balancer/ingress, r.RemoteAddr sẽ là IP của load balancer (chỉ 1
// giá trị duy nhất cho MỌI client) chứ không phải IP thật của client -
// X-Forwarded-For mới giữ được IP gốc. Ở giai đoạn hiện tại (docker-compose,
// gọi thẳng gateway) header này chưa có ai set, nên fallback về RemoteAddr.
func clientKey(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		// X-Forwarded-For co the la danh sach "client, proxy1, proxy2" -
		// IP dau tien la IP goc cua client.
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	return r.RemoteAddr
}
