package handlers

import (
	"database/sql"
	"net/http"
)

// HealthHandler cung cấp 2 endpoint riêng biệt cho 2 loại probe khác nhau
// trong k8s (chuẩn bị sẵn cho Lab 5.1):
//
//   - Healthz (liveness): chỉ trả lời "process này còn sống không". KHÔNG
//     kiểm tra DB, vì nếu Postgres down tạm thời mà liveness fail theo,
//     kubelet sẽ restart Pod liên tục dù chính app không hề bị treo -
//     việc restart đó không giải quyết được gì (DB vẫn down) mà còn gây
//     thêm nhiễu loạn.
//   - Ready (readiness): kiểm tra app có SẴN SÀNG nhận traffic không, tức
//     có kết nối được DB không. Nếu DB down, readiness fail -> k8s tự rút
//     Pod này khỏi Service endpoints, ngừng route traffic tới, nhưng
//     KHÔNG restart Pod - đúng hành vi mong muốn.
type HealthHandler struct {
	db *sql.DB
}

// NewHealthHandler tạo HealthHandler, nhận vào connection pool đã mở sẵn.
func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Healthz xử lý GET /healthz - dùng cho livenessProbe.
func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

// Ready xử lý GET /ready - dùng cho readinessProbe.
// Gọi Ping() để xác nhận connection pool vẫn nói chuyện được với Postgres.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database chua san sang: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
