package handlers

import (
	"net/http"

	"github.com/redis/go-redis/v9"
)

// SeatMapHandler trả lời câu hỏi "showtime này ghế nào đang bị giữ, ghế nào
// đã bán" - dùng cho frontend vẽ sơ đồ ghế (ghế nào tô xám/đỏ/xanh).
type SeatMapHandler struct {
	redisClient *redis.Client
}

func NewSeatMapHandler(redisClient *redis.Client) *SeatMapHandler {
	return &SeatMapHandler{redisClient: redisClient}
}

type seatMapResponse struct {
	Held   []string `json:"held"`   // dang bi giu tam, co the tro lai trong vai phut
	Booked []string `json:"booked"` // da ban vinh vien
}

// GetSeatMap xử lý GET /showtimes/{id}/seats.
//
// "held" lấy bằng cách SCAN các key theo pattern "hold:{showtime_id}:*".
// Dùng SCAN (con trỏ, quét dần) thay vì KEYS (quét 1 lần, block toàn bộ
// Redis) - KEYS có thể làm nghẽn Redis nếu số lượng key lớn, SCAN thì
// không, đây là thực hành chuẩn khi thao tác Redis trong hệ thống thật.
func (h *SeatMapHandler) GetSeatMap(w http.ResponseWriter, r *http.Request) {
	showtimeID := r.PathValue("id")
	ctx := r.Context()

	pattern := holdKey(showtimeID, "*")
	held := []string{}

	iter := h.redisClient.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		// key co dang "hold:{showtimeID}:{seat}" -> tach lay phan seat o cuoi.
		prefix := holdKey(showtimeID, "")
		seat := key[len(prefix):]
		held = append(held, seat)
	}
	if err := iter.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "loi quet redis: "+err.Error())
		return
	}

	booked, err := h.redisClient.SMembers(ctx, bookedSetKey(showtimeID)).Result()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "loi doc danh sach ghe da ban: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, seatMapResponse{Held: held, Booked: booked})
}
