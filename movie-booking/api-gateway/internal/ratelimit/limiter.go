// Package ratelimit cài đặt thuật toán "token bucket" đơn giản, giới hạn
// số request mỗi client (theo IP) được phép gửi trong 1 khoảng thời gian.
//
// NGUYÊN LÝ TOKEN BUCKET: mỗi client có 1 "xô" chứa tối đa `burst` token.
// Token được "rót" vào xô liên tục theo thời gian (tốc độ = requestsPerMinute
// / 60 token/giây). Mỗi request tiêu tốn 1 token; nếu xô rỗng (< 1 token),
// request bị từ chối (429). Xô đầy lại dần theo thời gian, không cần
// background job "reset mỗi phút" - chỉ cần tính "đã trôi qua bao lâu từ
// lần cuối, cộng thêm bấy nhiêu token" mỗi khi có request tới (lazy refill).
package ratelimit

import (
	"sync"
	"time"
)

type bucket struct {
	tokens     float64
	lastRefill time.Time
}

type RateLimiter struct {
	mu            sync.Mutex
	buckets       map[string]*bucket
	ratePerSecond float64
	burst         float64
}

// New tạo 1 RateLimiter cho phép tối đa requestsPerMinute request/phút cho
// MỖI client (phân biệt theo key, thường là địa chỉ IP).
func New(requestsPerMinute int) *RateLimiter {
	rl := &RateLimiter{
		buckets:       make(map[string]*bucket),
		ratePerSecond: float64(requestsPerMinute) / 60.0,
		burst:         float64(requestsPerMinute), // cho phep "dồn" toi da 1 phut request lien tuc
	}

	// Don dep dinh ky cac bucket lau khong dung, tranh map phinh to vo han
	// neu co rat nhieu IP khac nhau ghe qua (moi IP la 1 entry trong map,
	// khong bao gio bi xoa neu khong co buoc don dep nay).
	go rl.cleanupLoop()

	return rl
}

// Allow kiểm tra + tiêu tốn 1 token cho key này. Trả về false nếu đã vượt
// giới hạn (không còn token khả dụng tại thời điểm gọi).
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.buckets[key]
	if !exists {
		// Client moi -> tao bucket day (tru 1 token cho chinh request nay).
		rl.buckets[key] = &bucket{tokens: rl.burst - 1, lastRefill: now}
		return true
	}

	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * rl.ratePerSecond
	if b.tokens > rl.burst {
		b.tokens = rl.burst
	}
	b.lastRefill = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-10 * time.Minute)
		for key, b := range rl.buckets {
			if b.lastRefill.Before(cutoff) {
				delete(rl.buckets, key)
			}
		}
		rl.mu.Unlock()
	}
}
