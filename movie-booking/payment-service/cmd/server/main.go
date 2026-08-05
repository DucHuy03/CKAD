// payment-service - xử lý thanh toán (giả lập), publish event khi thành công.
//
// TỔNG QUAN LUỒNG CHẠY:
//
//  1. Đọc config (Postgres + RabbitMQ + URL booking-service).
//  2. Mở connection pool Postgres + connection/channel RabbitMQ.
//  3. Binary có 2 CHẾ ĐỘ CHẠY:
//     a) Mode "init" (-wait-db -wait-rabbitmq -migrate): chờ dependency
//     sẵn sàng, chạy migration, EXIT. -> Command của initContainer.
//     b) Mode "server" (mặc định): chạy HTTP server route /payments/*.
//     -> Command của container app.
//
// Không có mode "cleanup" ở đây (khác booking-service) vì payment-service
// không lưu dữ liệu nào cần tự động hết hạn - mỗi payment là 1 bản ghi
// lịch sử cố định, không có khái niệm "TTL" như booking HOLD.
package main

import (
	"database/sql"
	"flag"
	"log"
	"net/http"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"payment-service/internal/bookingclient"
	"payment-service/internal/config"
	"payment-service/internal/db"
	"payment-service/internal/handlers"
	"payment-service/internal/rabbitmq"
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

	// Mode server: can ket noi RabbitMQ that su (khong chi wait) de publish event.
	conn, ch, err := rabbitmq.Connect(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("khong the ket noi rabbitmq: %v", err)
	}
	defer conn.Close()
	defer ch.Close()

	if err := rabbitmq.DeclareExchange(ch); err != nil {
		log.Fatalf("khong the khai bao exchange: %v", err)
	}

	runServer(cfg, dbConn, ch)
}

func runServer(cfg config.Config, dbConn *sql.DB, ch *amqp.Channel) {
	mux := http.NewServeMux()

	bookingClient := bookingclient.New(cfg.BookingServiceURL)

	healthHandler := handlers.NewHealthHandler(dbConn, ch)
	paymentHandler := handlers.NewPaymentHandler(dbConn, ch, bookingClient)

	mux.HandleFunc("GET /healthz", healthHandler.Healthz)
	mux.HandleFunc("GET /ready", healthHandler.Ready)

	mux.HandleFunc("POST /payments/process", paymentHandler.Process)
	mux.HandleFunc("GET /payments/{id}", paymentHandler.GetByID)

	addr := ":" + strconv.Itoa(cfg.HTTPPort)
	log.Printf("payment-service dang lang nghe tai %s", addr)

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
