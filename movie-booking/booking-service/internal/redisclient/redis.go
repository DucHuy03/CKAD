// Package redisclient quản lý kết nối tới Redis - nơi lưu trạng thái
// "ghế nào đang bị giữ tạm" với TTL tự động hết hạn.
//
// TẠI SAO DÙNG REDIS CHO VIỆC NÀY thay vì chỉ Postgres:
//   - Redis có sẵn lệnh EXPIRE: set 1 key với TTL = 300s, sau 300s key tự
//     biến mất, không cần code nào đứng ra xoá. Với Postgres, muốn "tự hết
//     hạn" phải tự viết job quét định kỳ (đó là lý do bảng bookings trong
//     Postgres vẫn cần CronJob dọn dẹp ở Lab 1.3 - Postgres lưu LỊCH SỬ,
//     Redis mới là nguồn "sự thật" cho việc ghế có đang trống hay không).
//   - Redis rất nhanh cho check tồn tại 1 key (SETNX) - phù hợp cho tình
//     huống nhiều user cùng bấm giữ 1 ghế cùng lúc (race condition kinh
//     điển - SETNX ở Redis là atomic nên giải quyết gọn hơn nhiều so với
//     transaction + lock trong Postgres).
package redisclient

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"booking-service/internal/config"
)

// Connect tạo 1 Redis client theo cấu hình cfg.
// Giống sql.Open(), NewClient() không tự mở connection ngay (lazy) -
// muốn biết Redis có "sống" hay không phải gọi Ping().
func Connect(cfg config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
}

// WaitForRedis liên tục Ping() cho tới khi thành công hoặc hết timeout.
// Dùng trong initContainer giống hệt WaitForDB bên Postgres - Redis Pod
// cũng có thể chưa Ready khi booking-service Pod đã được k8s schedule.
func WaitForRedis(client *redis.Client, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		lastErr = client.Ping(ctx).Err()
		if lastErr == nil {
			return nil
		}
		log.Printf("cho redis san sang... (%v)", lastErr)
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("het thoi gian cho redis sau %s: %w", timeout, lastErr)
}
