// Package handlers chứa toàn bộ HTTP handler (tương đương "controller" trong
// các framework MVC). Mỗi handler chỉ lo 1 việc: đọc request, gọi query DB,
// trả JSON response. Không có business logic phức tạp ở movie-service vì
// đây chỉ là CRUD service, logic phức tạp (hold ghế, thanh toán...) nằm ở
// booking-service / payment-service.
package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

// errorResponse là format JSON thống nhất khi trả lỗi cho client,
// giúp api-gateway và các service khác parse lỗi theo 1 chuẩn duy nhất.
type errorResponse struct {
	Error string `json:"error"`
}

// writeJSON ghi bất kỳ giá trị Go nào ra response dưới dạng JSON, kèm status code.
// Tập trung logic set header + encode ở 1 chỗ để mọi handler gọi lại,
// tránh lặp code (và tránh quên set Content-Type ở handler nào đó).
func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		// Response header da flush roi nen khong the doi status code nua,
		// chi con cach log lai de debug qua kubectl logs.
		log.Printf("loi encode JSON response: %v", err)
	}
}

// writeError là helper rút gọn để trả lỗi kèm message, tự bọc vào errorResponse.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

// decodeJSON đọc JSON body của request vào struct dst.
// Trả về false (và đã tự ghi lỗi 400 ra response) nếu body không phải JSON hợp lệ,
// để handler gọi hàm này chỉ cần "if !decodeJSON(...) { return }".
func decodeJSON(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "request body khong phai JSON hop le: "+err.Error())
		return false
	}
	return true
}

// isNoRows kiểm tra lỗi trả về từ database/sql có phải "không tìm thấy dòng nào" không,
// dùng để phân biệt lỗi 404 (không có record) với 500 (lỗi DB thật sự).
func isNoRows(err error) bool {
	return err == sql.ErrNoRows
}
