package handlers

import (
	"database/sql"
	"net/http"

	"github.com/redis/go-redis/v9"
)

// HealthHandler ở booking-service khác movie-service 1 chỗ: readiness phải
// check CẢ Postgres LẪN Redis, vì booking-service phụ thuộc cả 2 - nếu chỉ
// 1 trong 2 down mà vẫn báo Ready thì traffic vẫn bị route vào Pod này và
// request sẽ fail ở tầng nghiệp vụ thay vì bị k8s tự động chặn từ tầng network.
type HealthHandler struct {
	db          *sql.DB
	redisClient *redis.Client
}

func NewHealthHandler(db *sql.DB, redisClient *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redisClient: redisClient}
}

func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "postgres chua san sang: "+err.Error())
		return
	}
	if err := h.redisClient.Ping(r.Context()).Err(); err != nil {
		writeError(w, http.StatusServiceUnavailable, "redis chua san sang: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
