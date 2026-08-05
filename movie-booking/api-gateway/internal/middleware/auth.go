// Package middleware chứa các "lớp bọc" (wrapper) quanh handler chính -
// mỗi middleware chỉ lo 1 việc (auth, rate limit...), ráp lại theo kiểu
// Chain of Responsibility: request đi qua từng lớp trước khi tới handler
// thật sự, mỗi lớp có quyền chặn request lại (trả lỗi luôn) hoặc cho đi tiếp.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"api-gateway/internal/auth"
)

// contextKey là type riêng (không phải string trần) để làm key khi lưu giá
// trị vào context.Context - tránh xung đột với key của package khác cũng
// lỡ dùng chuỗi "user_id" làm context key (Go vet sẽ cảnh báo nếu dùng
// string thường làm context key, đây là best practice chuẩn).
type contextKey string

const userIDContextKey contextKey = "user_id"

// RequireAuth trả về 1 middleware yêu cầu request phải có header
// "Authorization: Bearer <token>" hợp lệ mới được đi tiếp.
func RequireAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				writeUnauthorized(w, "thieu Authorization: Bearer <token>")
				return
			}

			tokenString := strings.TrimPrefix(header, "Bearer ")
			claims, err := auth.ValidateToken(jwtSecret, tokenString)
			if err != nil {
				writeUnauthorized(w, "token khong hop le: "+err.Error())
				return
			}

			// Nhet user_id vao context de handler phia sau (hoac chinh
			// backend service, neu gateway forward them header) co the dung.
			ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)

			// Forward luon user_id qua header rieng cho backend service -
			// vi backend service (booking-service, payment-service) hien tai
			// dang lay user_id tu BODY cua request (client tu khai bao), chua
			// co co che xac thuc rang user_id trong body co dung la nguoi
			// dang giu token hay khong. Day la 1 diem CAN CAI THIEN trong he
			// thong that: backend nen tin tuong header nay (do gateway them
			// vao SAU KHI xac thuc) hon la tin blindly vao body client gui len.
			r.Header.Set("X-User-ID", claims.UserID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"` + message + `"}`))
}
