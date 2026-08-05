package handlers

import (
	"database/sql"
	"net/http"

	amqp "github.com/rabbitmq/amqp091-go"
)

// HealthHandler readiness check cả Postgres lẫn RabbitMQ channel.
type HealthHandler struct {
	db *sql.DB
	ch *amqp.Channel
}

func NewHealthHandler(db *sql.DB, ch *amqp.Channel) *HealthHandler {
	return &HealthHandler{db: db, ch: ch}
}

func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "postgres chua san sang: "+err.Error())
		return
	}
	// amqp091-go khong co ham Ping() cho channel, nhung channel bi dong
	// (IsClosed() == true) neu connection toi RabbitMQ da mat - kiem tra
	// co ban nay du de readinessProbe phat hien truong hop RabbitMQ down.
	if h.ch.IsClosed() {
		writeError(w, http.StatusServiceUnavailable, "ket noi rabbitmq da dong")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
