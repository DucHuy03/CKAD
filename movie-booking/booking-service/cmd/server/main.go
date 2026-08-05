// booking-service - chọn ghế, giữ ghế tạm thời (TTL qua Redis), confirm booking.
//
// TỔNG QUAN LUỒNG CHẠY:
//
//  1. Đọc config (Postgres + Redis + URL của movie-service).
//
//  2. Mở connection pool Postgres + Redis client.
//
//  3. Binary này có 3 CHẾ ĐỘ CHẠY (nhiều hơn movie-service 1 mode, vì
//     nghiệp vụ "giữ ghế có TTL" cần thêm 1 job dọn dẹp định kỳ):
//
//     a) Mode "init" (-wait-db -wait-redis -migrate):
//     Chờ Postgres + Redis sẵn sàng, chạy migration, EXIT.
//     -> Command của initContainer.
//
//     b) Mode "server" (mặc định):
//     Chạy HTTP server, các route /bookings/*, /showtimes/{id}/seats.
//     -> Command của container app.
//
//     c) Mode "cleanup" (-cleanup-expired-holds):
//     Quét Postgres tìm các booking status=HOLD đã quá expires_at,
//     chuyển sang EXPIRED, dọn luôn hold key còn sót trong Redis (đề
//     phòng TTL không khớp do lệch giờ hoặc test tay), rồi EXIT.
//     -> Command của CronJob (Lab 1.3) - chạy định kỳ mỗi vài phút.
//
// Lưu ý quan trọng: Redis TTL là nguồn "sự thật" quyết định ghế có đang bị
// khoá hay không theo thời gian thực. Mode cleanup KHÔNG ảnh hưởng tới việc
// đó - nó chỉ dọn lại bản ghi trong Postgres cho đúng thực tế, để booking
// không bị "kẹt" mãi ở trạng thái HOLD trong lịch sử dù ghế đã được thả từ lâu.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"booking-service/internal/config"
	"booking-service/internal/db"
	"booking-service/internal/handlers"
	"booking-service/internal/movieclient"
	"booking-service/internal/redisclient"
)

func main() {
	waitDB := flag.Bool("wait-db", false, "cho Postgres san sang (dung cho init container)")
	waitRedis := flag.Bool("wait-redis", false, "cho Redis san sang (dung cho init container)")
	migrate := flag.Bool("migrate", false, "chay migration SQL roi thoat (dung cho init container)")
	cleanup := flag.Bool("cleanup-expired-holds", false, "quet va danh dau cac booking HOLD da het han (dung cho CronJob)")
	migrationsDir := flag.String("migrations-dir", "migrations", "duong dan thu muc migration")
	flag.Parse()

	cfg := config.Load()

	dbConn, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("khong the mo connection pool postgres: %v", err)
	}
	defer dbConn.Close()

	redisClient := redisclient.Connect(cfg)
	defer redisClient.Close()

	if *waitDB {
		log.Println("dang cho postgres san sang...")
		if err := db.WaitForDB(dbConn, 60*time.Second); err != nil {
			log.Fatalf("postgres khong san sang: %v", err)
		}
		log.Println("postgres da san sang")
	}

	if *waitRedis {
		log.Println("dang cho redis san sang...")
		if err := redisclient.WaitForRedis(redisClient, 60*time.Second); err != nil {
			log.Fatalf("redis khong san sang: %v", err)
		}
		log.Println("redis da san sang")
	}

	if *migrate {
		log.Println("dang chay migration...")
		if err := db.RunMigrations(dbConn, *migrationsDir); err != nil {
			log.Fatalf("chay migration that bai: %v", err)
		}
		log.Println("migration hoan tat, thoat (init container xong nhiem vu)")
		return
	}

	if *cleanup {
		if err := runCleanupExpiredHolds(dbConn, redisClient); err != nil {
			log.Fatalf("cleanup that bai: %v", err)
		}
		return
	}

	runServer(cfg, dbConn, redisClient)
}

// runCleanupExpiredHolds la logic ma CronJob (Lab 1.3) se goi dinh ky.
//
// FLOW: SELECT cac booking status=HOLD va expires_at < now() -> voi moi
// booking, xoa cac hold key lien quan trong Redis (don dep phong ho, vi
// binh thuong TTL da tu xoa roi) -> UPDATE status thanh EXPIRED trong Postgres.
func runCleanupExpiredHolds(dbConn *sql.DB, redisClient *redis.Client) error {
	ctx := context.Background()

	rows, err := dbConn.QueryContext(ctx, `
		SELECT id, showtime_id, seats
		FROM bookings
		WHERE status = 'HOLD' AND expires_at < now()
	`)
	if err != nil {
		return fmt.Errorf("truy van booking het han that bai: %w", err)
	}

	type expiredBooking struct {
		id         string
		showtimeID string
		seats      pq.StringArray
	}
	var expired []expiredBooking
	for rows.Next() {
		var b expiredBooking
		if err := rows.Scan(&b.id, &b.showtimeID, &b.seats); err != nil {
			rows.Close()
			return fmt.Errorf("doc du lieu booking het han that bai: %w", err)
		}
		expired = append(expired, b)
	}
	rows.Close()

	log.Printf("tim thay %d booking HOLD da het han", len(expired))

	for _, b := range expired {
		// Don dep phong ho: neu Redis TTL vi ly do gi chua kip xoa (VD lech gio),
		// chu dong xoa lai hold key cho tung ghe.
		for _, seat := range b.seats {
			redisClient.Del(ctx, "hold:"+b.showtimeID+":"+seat)
		}

		if _, err := dbConn.ExecContext(ctx, `
			UPDATE bookings SET status = 'EXPIRED', updated_at = now() WHERE id = $1
		`, b.id); err != nil {
			log.Printf("canh bao: cap nhat EXPIRED cho booking %s that bai: %v", b.id, err)
			continue
		}
		log.Printf("da danh dau booking %s la EXPIRED", b.id)
	}

	return nil
}

func runServer(cfg config.Config, dbConn *sql.DB, redisClient *redis.Client) {
	mux := http.NewServeMux()

	movieClient := movieclient.New(cfg.MovieServiceURL)

	healthHandler := handlers.NewHealthHandler(dbConn, redisClient)
	bookingHandler := handlers.NewBookingHandler(dbConn, redisClient, movieClient, cfg)
	seatMapHandler := handlers.NewSeatMapHandler(redisClient)

	mux.HandleFunc("GET /healthz", healthHandler.Healthz)
	mux.HandleFunc("GET /ready", healthHandler.Ready)

	mux.HandleFunc("POST /bookings/hold", bookingHandler.Hold)
	mux.HandleFunc("POST /bookings/{id}/confirm", bookingHandler.Confirm)
	mux.HandleFunc("POST /bookings/{id}/cancel", bookingHandler.Cancel)
	mux.HandleFunc("GET /bookings/{id}", bookingHandler.GetByID)

	mux.HandleFunc("GET /showtimes/{id}/seats", seatMapHandler.GetSeatMap)

	addr := ":" + strconv.Itoa(cfg.HTTPPort)
	log.Printf("booking-service dang lang nghe tai %s", addr)

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("http server dung dot ngot: %v", err)
	}
}
