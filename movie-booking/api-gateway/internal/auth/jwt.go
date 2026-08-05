// Package auth xử lý việc tạo và xác minh JWT (JSON Web Token) - cơ chế
// xác thực stateless: api-gateway không cần lưu session ở đâu cả, mọi
// thông tin cần thiết (user_id, thời hạn) nằm ngay trong token, chữ ký
// HMAC đảm bảo token không bị giả mạo (không ai sửa được payload mà không
// biết JWT_SECRET để ký lại đúng).
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims là dữ liệu được nhúng vào token. Nhúng thêm jwt.RegisteredClaims
// để có sẵn các field chuẩn (exp, iat...) mà thư viện tự validate giúp
// (ví dụ tự động báo lỗi nếu token đã hết hạn, không cần code tự so sánh).
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateToken tạo 1 JWT ký bằng HMAC-SHA256, chứa user_id và thời hạn hết hạn.
func GenerateToken(secret, userID string, expiry time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken xác minh chữ ký + thời hạn của token, trả về Claims nếu hợp lệ.
func ValidateToken(secret, tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		// Kiem tra ro rang thuat toan ky la HMAC - tranh 1 lop tan cong JWT
		// kinh dien: ke tan cong doi header "alg" sang "none" hoac sang RSA
		// (neu server dung public key de "verify" ma khong check thuat toan,
		// ke tan cong co the tu ky bang chinh public key do). Luon ep kieu
		// va kiem tra thuat toan minh mong doi truoc khi tra ve key.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("thuat toan ky khong hop le: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("token khong hop le: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("token khong hop le")
	}

	return claims, nil
}
