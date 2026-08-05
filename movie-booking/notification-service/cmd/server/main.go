// notification-service - consume event "payment.success" từ RabbitMQ, gửi
// email kèm mã QR vé.
//
// TỔNG QUAN LUỒNG CHẠY - ĐIỂM KHÁC BIỆT LỚN NHẤT SO VỚI 3 SERVICE TRƯỚC:
// Cả 3 service trước (movie/booking/payment) đều là "thuần request-response"
// - có request tới thì mới làm việc. notification-service THÊM 1 luồng xử
// lý CHỦ ĐỘNG chạy nền: ngay khi service start (mode server), nó tự mở 1
// goroutine gọi consumer.Run() để liên tục lắng nghe RabbitMQ, KHÔNG cần ai
// gọi HTTP tới nó cả. HTTP server ở service này chỉ đóng vai trò phụ
// (health check + tra cứu lịch sử).
//
//  1. Đọc config (Postgres + RabbitMQ + SMTP).
//  2. Mở connection pool Postgres.
//  3. Binary có 2 mode:
//     a) Mode "init" (-wait-db -wait-rabbitmq -migrate): giống các service trước.
//     b) Mode "server" (mặc định):
//     - Mở connection + channel RabbitMQ, khai báo queue + bind exchange
//     - Chạy "go consumer.Run(ch)" ở 1 goroutine riêng (KHÔNG block main)
//     - Chạy HTTP server ở goroutine chính (block như bình thường)
package main

import (
	"database/sql"
	"flag"
	"log"
	"net/http"
	"strconv"
	"time"

	"notification-service/internal/config"
	"notification-service/internal/consumer"
	"notification-service/internal/db"
	"notification-service/internal/handlers"
	"notification-service/internal/mailer"
	"notification-service/internal/rabbitmq"
)

func main() {
	waitDB := flag.Bool("wait-db", false, "cho Postgres san sang (dung cho init container)")
	waitRabbitMQ := flag.Bool("wait-rabbitmq", false, "cho RabbitMQ san sang (dung cho init container)")
	migrate := flag.Bool("migrate", false, "chay migration SQL roi thoat (dung cho init container)")
	migrationsDir := flag.String("migrations-dir", "migrations", "duong dan thu muc migration")
	flag.Parse()

	cfg := config.Load()

	dbConn, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("khong the mo connection pool postgres: %v", err)
	}
	defer dbConn.Close()

	if *waitDB {
		log.Println("dang cho postgres san sang...")
		if err := db.WaitForDB(dbConn, 60*time.Second); err != nil {
			log.Fatalf("postgres khong san sang: %v", err)
		}
		log.Println("postgres da san sang")
	}

	if *waitRabbitMQ {
		log.Println("dang cho rabbitmq san sang...")
		if err := rabbitmq.WaitForRabbitMQ(cfg.RabbitMQURL, 60*time.Second); err != nil {
			log.Fatalf("rabbitmq khong san sang: %v", err)
		}
		log.Println("rabbitmq da san sang")
	}

	if *migrate {
		log.Println("dang chay migration...")
		if err := db.RunMigrations(dbConn, *migrationsDir); err != nil {
			log.Fatalf("chay migration that bai: %v", err)
		}
		log.Println("migration hoan tat, thoat (init container xong nhiem vu)")
		return
	}

	// --- Mode server: mo ket noi RabbitMQ that su + khoi dong consumer nen ---
	conn, ch, err := rabbitmq.Connect(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("khong the ket noi rabbitmq: %v", err)
	}
	defer conn.Close()
	defer ch.Close()

	if err := rabbitmq.SetupConsumerTopology(ch); err != nil {
		log.Fatalf("khong the thiet lap queue/binding: %v", err)
	}

	mailerCfg := mailer.Config{Host: cfg.SMTPHost, Port: cfg.SMTPPort, From: cfg.MailFrom}
	c := consumer.New(dbConn, mailerCfg)

	// QUAN TRONG: chay bang goroutine ("go ..."), khong phai goi truc tiep
	// c.Run(ch) - vi Run() block mai mai, neu goi truc tiep (khong co "go")
	// thi dong lenh runServer() ben duoi se KHONG BAO GIO duoc thuc thi.
	go c.Run(ch)

	runServer(cfg, dbConn)
}

func runServer(cfg config.Config, dbConn *sql.DB) {
	mux := http.NewServeMux()

	healthHandler := handlers.NewHealthHandler(dbConn)
	notificationHandler := handlers.NewNotificationHandler(dbConn)

	mux.HandleFunc("GET /healthz", healthHandler.Healthz)
	mux.HandleFunc("GET /ready", healthHandler.Ready)
	mux.HandleFunc("GET /notifications/by-booking/{booking_id}", notificationHandler.GetByBookingID)

	addr := ":" + strconv.Itoa(cfg.HTTPPort)
	log.Printf("notification-service dang lang nghe tai %s", addr)

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
