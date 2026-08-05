package handlers

import "net/http"

// HealthHandler ở api-gateway đơn giản hơn hẳn các service trước vì gateway
// KHÔNG có DB/Redis/RabbitMQ riêng để check - nó chỉ chuyển tiếp request.
// Ready trả OK ngay (gateway sẵn sàng nhận request) - việc backend phía sau
// có sống hay không là trách nhiệm của initContainer (chờ /healthz của cả 4
// backend TRƯỚC KHI container app này được phép start), không phải của
// readinessProbe tại runtime - nếu readiness phụ thuộc backend, 1 backend
// down sẽ kéo theo gateway bị rút khỏi Service endpoints dù bản thân gateway
// vẫn hoàn toàn khoẻ mạnh và có thể phục vụ các route không liên quan.
type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
