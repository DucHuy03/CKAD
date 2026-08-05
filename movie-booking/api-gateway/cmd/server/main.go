// api-gateway - entry point duy nhất mà client bên ngoài gọi vào, route
// request tới đúng backend, xác thực JWT, giới hạn tốc độ request.
//
// TỔNG QUAN LUỒNG CHẠY:
//
//  1. Đọc config (URL của 4 backend, JWT secret, ngưỡng rate limit).
//  2. Binary có 2 CHẾ ĐỘ CHẠY:
//     a) Mode "init" (-wait-backends): POLL /healthz của CẢ 4 backend
//     (movie/booking/payment/notification-service) cho tới khi tất cả
//     đều trả 200 hoặc hết timeout, rồi EXIT. Đây là initContainer của
//     api-gateway - khác 4 service trước (chờ DB/Redis/RabbitMQ), ở đây
//     "dependency" chính là CÁC SERVICE KHÁC, vì gateway không có ý
//     nghĩa gì nếu backend chưa sẵn sàng.
//     b) Mode "server" (mặc định): khởi động HTTP server, đăng ký route.
//
// KIẾN TRÚC MIDDLEWARE (áp dụng theo thứ tự với MỖI route):
//
//	request -> [RateLimit] -> [RequireAuth (chỉ áp cho route cần)] -> [Proxy tới backend]
//
// RateLimit áp dụng CHO TẤT CẢ route (kể cả /auth/login) để chống brute-force
// và chống 1 client duy nhất làm nghẽn toàn hệ thống. RequireAuth chỉ áp cho
// các route có tác động dữ liệu (đặt vé, thanh toán...) - route xem phim/rạp/
// suất chiếu (browsing) cố tình để công khai, không cần đăng nhập mới xem được.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"strconv"
	"time"

	"api-gateway/internal/config"
	"api-gateway/internal/handlers"
	"api-gateway/internal/middleware"
	"api-gateway/internal/proxy"
	"api-gateway/internal/ratelimit"
)

func main() {
	waitBackends := flag.Bool("wait-backends", false, "cho 4 backend service /healthz san sang roi thoat (dung cho init container)")
	flag.Parse()

	cfg := config.Load()

	if *waitBackends {
		if err := waitForBackends(cfg, 60*time.Second); err != nil {
			log.Fatalf("cac backend service khong san sang: %v", err)
		}
		log.Println("tat ca backend da san sang, thoat (init container xong nhiem vu)")
		return
	}

	runServer(cfg)
}

// waitForBackends poll GET {baseURL}/healthz của từng backend cho tới khi
// TẤT CẢ đều trả 200, hoặc hết timeout thì trả lỗi.
func waitForBackends(cfg config.Config, timeout time.Duration) error {
	backends := map[string]string{
		"movie-service":        cfg.MovieServiceURL,
		"booking-service":      cfg.BookingServiceURL,
		"payment-service":      cfg.PaymentServiceURL,
		"notification-service": cfg.NotificationServiceURL,
	}

	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 3 * time.Second}

	for name, baseURL := range backends {
		ok := false
		var lastErr error

		for time.Now().Before(deadline) {
			resp, err := client.Get(baseURL + "/healthz")
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				ok = true
				break
			}
			if resp != nil {
				resp.Body.Close()
			}
			lastErr = err
			log.Printf("cho %s san sang tai %s...", name, baseURL)
			time.Sleep(2 * time.Second)
		}

		if !ok {
			return fmt.Errorf("%s khong san sang sau %s: %v", name, timeout, lastErr)
		}
		log.Printf("%s da san sang", name)
	}

	return nil
}

func runServer(cfg config.Config) {
	mux := http.NewServeMux()

	// --- Khoi tao proxy toi tung backend ---
	movieProxy := mustNewProxy(cfg.MovieServiceURL)
	bookingProxy := mustNewProxy(cfg.BookingServiceURL)
	paymentProxy := mustNewProxy(cfg.PaymentServiceURL)
	notificationProxy := mustNewProxy(cfg.NotificationServiceURL)

	authHandler := handlers.NewAuthHandler(cfg)
	healthHandler := handlers.NewHealthHandler()

	// --- Route noi bo cua gateway (khong proxy di dau) ---
	mux.HandleFunc("GET /healthz", healthHandler.Healthz)
	mux.HandleFunc("GET /ready", healthHandler.Ready)
	mux.HandleFunc("POST /auth/login", authHandler.Login)

	authMW := middleware.RequireAuth(cfg.JWTSecret)

	// --- Route CONG KHAI (browsing phim/rap/suat chieu - khong can dang nhap) ---
	mux.Handle("GET /api/movies", movieProxy)
	mux.Handle("GET /api/movies/{id}", movieProxy)
	mux.Handle("GET /api/cinemas", movieProxy)
	mux.Handle("GET /api/cinemas/{id}", movieProxy)
	mux.Handle("GET /api/showtimes", movieProxy)
	mux.Handle("GET /api/showtimes/{id}", movieProxy)
	mux.Handle("GET /api/showtimes/{id}/seats", bookingProxy) // luu y: /seats thuc te nam o booking-service, khong phai movie-service

	// --- Route quan tri phim/rap/suat chieu - CAN dang nhap (dang le nen la role admin, nhung chua co RBAC o muc bai lab nay) ---
	mux.Handle("POST /api/movies", authMW(movieProxy))
	mux.Handle("PUT /api/movies/{id}", authMW(movieProxy))
	mux.Handle("DELETE /api/movies/{id}", authMW(movieProxy))
	mux.Handle("POST /api/cinemas", authMW(movieProxy))
	mux.Handle("POST /api/showtimes", authMW(movieProxy))

	// --- Route booking - CAN dang nhap (thao tac gan voi 1 user cu the) ---
	mux.Handle("POST /api/bookings/hold", authMW(bookingProxy))
	mux.Handle("POST /api/bookings/{id}/confirm", authMW(bookingProxy))
	mux.Handle("POST /api/bookings/{id}/cancel", authMW(bookingProxy))
	mux.Handle("GET /api/bookings/{id}", authMW(bookingProxy))

	// --- Route payment - CAN dang nhap ---
	mux.Handle("POST /api/payments/process", authMW(paymentProxy))
	mux.Handle("GET /api/payments/{id}", authMW(paymentProxy))

	// --- Route notification - CAN dang nhap (du lieu ca nhan) ---
	mux.Handle("GET /api/notifications/by-booking/{booking_id}", authMW(notificationProxy))

	// --- Boc toan bo mux bang RateLimit (ap dung cho MOI route, ke ca /auth/login) ---
	limiter := ratelimit.New(cfg.RateLimitPerMinute)
	handler := middleware.RateLimit(limiter)(mux)

	addr := ":" + strconv.Itoa(cfg.HTTPPort)
	log.Printf("api-gateway dang lang nghe tai %s", addr)
	log.Printf("rate limit: %d request/phut/client", cfg.RateLimitPerMinute)

	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second, // dai hon cac service khac vi request co the phai doi backend xu ly qua proxy
		IdleTimeout:  60 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("http server dung dot ngot: %v", err)
	}
}

func mustNewProxy(targetBase string) *httputil.ReverseProxy {
	p, err := proxy.New(targetBase)
	if err != nil {
		log.Fatalf("khoi tao proxy toi %q that bai: %v", targetBase, err)
	}
	return p
}
