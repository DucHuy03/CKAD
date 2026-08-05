// movie-service - CRUD service cho phim, rạp, suất chiếu (showtime).
//
// TỔNG QUAN LUỒNG CHẠY CỦA CHƯƠNG TRÌNH NÀY:
//
//  1. Đọc cấu hình từ biến môi trường (package config) - trong docker-compose
//     là các biến khai báo trong file compose, trong k8s là ConfigMap + Secret.
//
//  2. Mở connection pool tới Postgres (package db).
//
//  3. Binary này có 2 CHẾ ĐỘ CHẠY, chọn bằng flag dòng lệnh, để cùng 1 image
//     Docker dùng được cho cả 2 container khác nhau trong Pod (Lab 1.2):
//
//     a) Chế độ "init" (-wait-db và/hoặc -migrate):
//     ./server -wait-db -migrate
//     -> Chờ Postgres sẵn sàng, chạy migration SQL, rồi EXIT NGAY (exit 0).
//     Đây là command của initContainer - initContainer không chạy HTTP
//     server, chỉ làm xong việc chuẩn bị rồi kết thúc để k8s cho phép
//     app container start.
//
//     b) Chế độ "server" (mặc định, không truyền flag nào):
//     ./server
//     -> Khởi động HTTP server, lắng nghe route /healthz, /ready,
//     /movies, /cinemas, /showtimes. Đây là command của app container,
//     chạy mãi cho tới khi Pod bị dừng.
//
// Tách 2 chế độ như vậy giúp ta KHÔNG cần build 2 image riêng cho init
// container và app container - giảm số lượng image cần quản lý, đồng thời
// đảm bảo logic connect DB (DSN, retry...) chỉ viết 1 lần, dùng chung.
package main

import (
	"database/sql"
	"flag"
	"log"
	"net/http"
	"strconv"
	"time"

	"movie-service/internal/applog"
	"movie-service/internal/config"
	"movie-service/internal/db"
	"movie-service/internal/handlers"
)

func main() {
	// --- Bước 0: thiết lập logging SỚM NHẤT (trước cả đọc config chi tiết
	// khác), để mọi log.Printf từ đây trở đi đều ghi đúng nơi (stdout + file
	// nếu LOG_FILE_PATH được set) ---
	cfg := config.Load()
	if logFile, err := applog.Setup(cfg.LogFilePath); err != nil {
		log.Fatalf("khong the thiet lap logging: %v", err)
	} else if logFile != nil {
		defer logFile.Close()
	}

	// --- Bước 1: đọc flag dòng lệnh để biết chương trình đang chạy ở mode nào ---
	waitDB := flag.Bool("wait-db", false, "cho Postgres san sang truoc khi lam gi tiep (dung cho init container)")
	migrate := flag.Bool("migrate", false, "chay migration SQL trong thu muc migrations/ roi thoat (dung cho init container)")
	migrationsDir := flag.String("migrations-dir", "migrations", "duong dan thu muc chua file migration .sql")
	flag.Parse()

	// --- Bước 2: mở connection pool (dùng chung cho mọi mode) ---
	dbConn, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("khong the mo connection pool: %v", err)
	}
	defer dbConn.Close()

	// --- Bước 3: nếu được yêu cầu, chờ DB sẵn sàng trước khi làm bất cứ gì khác ---
	if *waitDB {
		log.Println("dang cho postgres san sang...")
		if err := db.WaitForDB(dbConn, 60*time.Second); err != nil {
			log.Fatalf("postgres khong san sang sau thoi gian cho: %v", err)
		}
		log.Println("postgres da san sang")
	}

	// --- Bước 4: nếu được yêu cầu, chạy migration rồi EXIT (không chạy server) ---
	// Đây là nhánh mà initContainer sẽ đi vào: làm xong migration là kết thúc process,
	// container init trong Pod chuyển sang trạng thái "Completed", app container mới được start.
	if *migrate {
		log.Println("dang chay migration...")
		if err := db.RunMigrations(dbConn, *migrationsDir); err != nil {
			log.Fatalf("chay migration that bai: %v", err)
		}
		log.Println("migration hoan tat, thoat chuong trinh (init container xong nhiem vu)")
		return // exit 0 - quan trong: initContainer phai exit code 0 thi Pod moi chuyen tiep
	}

	// --- Bước 5: mode server bình thường - khởi động HTTP server ---
	runServer(cfg, dbConn)
}

// runServer đăng ký toàn bộ route và khởi động HTTP server, block cho tới
// khi có lỗi nghiêm trọng (ListenAndServe chỉ return khi server dừng).
func runServer(cfg config.Config, dbConn *sql.DB) {
	mux := http.NewServeMux()

	healthHandler := handlers.NewHealthHandler(dbConn)
	movieHandler := handlers.NewMovieHandler(dbConn)
	cinemaHandler := handlers.NewCinemaHandler(dbConn)
	showtimeHandler := handlers.NewShowtimeHandler(dbConn)

	// Health & readiness - dung cho k8s probes (Lab 5.1)
	mux.HandleFunc("GET /healthz", healthHandler.Healthz)
	mux.HandleFunc("GET /ready", healthHandler.Ready)

	// Movies CRUD
	mux.HandleFunc("POST /movies", movieHandler.Create)
	mux.HandleFunc("GET /movies", movieHandler.List)
	mux.HandleFunc("GET /movies/{id}", movieHandler.GetByID)
	mux.HandleFunc("PUT /movies/{id}", movieHandler.Update)
	mux.HandleFunc("DELETE /movies/{id}", movieHandler.Delete)

	// Cinemas (Create/List/GetByID)
	mux.HandleFunc("POST /cinemas", cinemaHandler.Create)
	mux.HandleFunc("GET /cinemas", cinemaHandler.List)
	mux.HandleFunc("GET /cinemas/{id}", cinemaHandler.GetByID)

	// Showtimes (Create/List co filter/GetByID)
	mux.HandleFunc("POST /showtimes", showtimeHandler.Create)
	mux.HandleFunc("GET /showtimes", showtimeHandler.List)
	mux.HandleFunc("GET /showtimes/{id}", showtimeHandler.GetByID)

	addr := ":" + strconv.Itoa(cfg.HTTPPort)
	log.Printf("movie-service dang lang nghe tai %s", addr)

	// Set timeout hop ly de tranh 1 client cham/treo lam nghen ca server
	// (quan trong voi he thong that, du bai lab khong test truong hop nay).
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
