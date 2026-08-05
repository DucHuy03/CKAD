package handlers

import (
	"database/sql"
	"net/http"
)

type HealthHandler struct {
	db *sql.DB
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

// Ready chỉ check Postgres. Không check trực tiếp RabbitMQ connection của
// consumer ở đây để giữ handler đơn giản (độc lập với goroutine consumer) -
// nếu muốn giám sát chặt hơn tình trạng consumer, có thể thêm 1 biến global
// "lastMessageProcessedAt" và cảnh báo nếu quá lâu không có message nào (dù
// cách này dễ gây false positive nếu đơn giản là không có giao dịch nào mới).
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "postgres chua san sang: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
