// Package applog cấu hình output của package "log" chuẩn: ghi ĐỒNG THỜI ra
// stdout VÀ 1 file trên đĩa.
//
// TẠI SAO GHI CẢ 2 NƠI (không chỉ ghi file):
//   - stdout: để `docker logs`/`kubectl logs <pod> -c app` xem được NGAY LẬP
//     TỨC, không phụ thuộc sidecar log-shipper có đang chạy hay không -
//     nếu chỉ ghi file mà sidecar bị lỗi/chưa start, ta sẽ mất khả năng
//     debug qua log trong lúc đó.
//   - file: để sidecar log-shipper (nằm trong CÙNG Pod, chia sẻ volume
//     emptyDir) đọc được và tail riêng qua `kubectl logs <pod> -c log-shipper`
//   - đây là yêu cầu của Lab 1.2.
//
// Nếu path rỗng (biến môi trường LOG_FILE_PATH không được set - trường hợp
// chạy local không có sidecar), chỉ ghi ra stdout như hành vi mặc định của
// package log, KHÔNG báo lỗi gì cả - service vẫn hoạt động bình thường.
package applog

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// Setup mở (hoặc tạo mới) file log tại path, chuyển output của package log
// sang ghi đồng thời stdout + file đó. Trả về *os.File để caller tự defer
// Close() khi chương trình kết thúc (đảm bảo flush hết dữ liệu còn đệm).
func Setup(path string) (*os.File, error) {
	if path == "" {
		return nil, nil // khong cau hinh file -> giu nguyen hanh vi mac dinh (chi stdout)
	}

	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("khong tao duoc thu muc chua file log %q: %w", dir, err)
		}
	}

	// O_APPEND: neu file da ton tai tu lan chay truoc (container restart nhung
	// volume emptyDir van con - thuc te emptyDir mat khi Pod bi xoa, nhung
	// container restart trong CUNG Pod thi volume van con), ghi noi tiep chu
	// khong ghi de, tranh mat log cu 1 cach dot ngot.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("khong mo duoc file log %q: %w", path, err)
	}

	log.SetOutput(io.MultiWriter(os.Stdout, f))
	log.Printf("applog: dang ghi log dong thoi ra stdout va file %q", path)

	return f, nil
}
