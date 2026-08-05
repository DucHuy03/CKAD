package handlers

import (
	"net/http"

	"api-gateway/internal/auth"
	"api-gateway/internal/config"
)

type AuthHandler struct {
	cfg config.Config
}

func NewAuthHandler(cfg config.Config) *AuthHandler {
	return &AuthHandler{cfg: cfg}
}

type loginRequest struct {
	UserID   string `json:"user_id"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in_seconds"`
}

// Login xử lý POST /auth/login.
//
// LƯU Ý QUAN TRỌNG: hệ thống này CHƯA có user-service quản lý tài khoản
// thật, nên endpoint này KHÔNG kiểm tra password (chấp nhận bất kỳ giá trị
// nào) - chỉ cần user_id không rỗng là issue token ngay. Đây CHỈ phù hợp
// cho mục đích học tập/demo luồng JWT + middleware auth; hệ thống thật bắt
// buộc phải có bước xác thực credential (so khớp password hash trong DB,
// hoặc gọi sang 1 identity provider) trước khi issue token.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, "user_id khong duoc rong")
		return
	}

	token, err := auth.GenerateToken(h.cfg.JWTSecret, req.UserID, h.cfg.JWTExpiryDuration())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "tao token that bai: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{
		Token:     token,
		ExpiresIn: h.cfg.JWTExpiryMins * 60,
	})
}
