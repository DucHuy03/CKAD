// Package db xử lý mọi thứ liên quan tới kết nối và khởi tạo schema Postgres.
//
// 3 hàm chính trong file này ánh xạ trực tiếp tới 3 việc mà initContainer
// cần làm trước khi app container được phép start (xem Phase 4 - Lab 1.2):
//  1. Connect      -> mở connection pool
//  2. WaitForDB    -> retry cho tới khi Postgres nhận connection (DB có thể
//     start chậm hơn app do là Pod/container khác)
//  3. RunMigrations -> áp toàn bộ file .sql trong thư mục migrations/
package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	// Import driver Postgres theo side-effect (dấu "_"): package tự đăng ký
	// tên driver "postgres" vào database/sql qua hàm init(), ta không gọi
	// trực tiếp hàm nào từ package này, chỉ cần nó "có mặt".
	_ "github.com/lib/pq"

	"movie-service/internal/config"
)

// Connect mở một connection pool tới Postgres theo cấu hình cfg.
//
// Lưu ý: sql.Open() KHÔNG thực sự mở connection ngay (lazy), nó chỉ validate
// DSN. Muốn biết DB có "sống" hay không phải gọi Ping() — đó là lý do
// WaitForDB() bên dưới tồn tại riêng.
func Connect(cfg config.Config) (*sql.DB, error) {
	dbConn, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("mo ket noi postgres that bai: %w", err)
	}

	// Giới hạn pool để 1 Pod không mở tràn lan connection làm quá tải Postgres
	// khi có nhiều replica movie-service cùng chạy.
	dbConn.SetMaxOpenConns(10)
	dbConn.SetMaxIdleConns(5)
	dbConn.SetConnMaxLifetime(5 * time.Minute)

	return dbConn, nil
}

// WaitForDB liên tục thử Ping() cho tới khi thành công hoặc hết timeout.
//
// Đây chính là logic mà initContainer sẽ chạy: Postgres Pod có thể chưa
// Ready khi app Pod đã được schedule (không có thứ tự đảm bảo giữa các
// Deployment độc lập trong k8s) -> nếu app container start ngay và connect
// thất bại thì sẽ bị CrashLoopBackOff. Init container "chờ hộ" nên app
// container chỉ start khi chắc chắn DB đã nhận connection.
func WaitForDB(dbConn *sql.DB, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		lastErr = dbConn.Ping()
		if lastErr == nil {
			return nil // DB da san sang
		}
		log.Printf("cho postgres san sang... (%v)", lastErr)
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("het thoi gian cho postgres sau %s: %w", timeout, lastErr)
}

// RunMigrations đọc toàn bộ file *.sql trong migrationsDir, sắp xếp theo tên
// (001_init.sql chạy trước 002_xxx.sql...) và thực thi tuần tự.
//
// Đây là bản migration tối giản (không có bảng theo dõi version) — đủ dùng
// cho mục đích học tập vì mọi file SQL đều viết idempotent (CREATE TABLE IF
// NOT EXISTS). Với hệ thống production thật, nên thay bằng công cụ chuyên
// dụng như golang-migrate để track version + hỗ trợ rollback.
func RunMigrations(dbConn *sql.DB, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("khong doc duoc thu muc migrations %q: %w", migrationsDir, err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sql" {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files) // dam bao thu tu 001_, 002_, ...

	if len(files) == 0 {
		log.Printf("khong co file migration nao trong %q", migrationsDir)
		return nil
	}

	for _, fname := range files {
		fullPath := filepath.Join(migrationsDir, fname)
		sqlBytes, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("khong doc duoc file migration %q: %w", fullPath, err)
		}

		log.Printf("dang chay migration: %s", fname)
		if _, err := dbConn.Exec(string(sqlBytes)); err != nil {
			return fmt.Errorf("migration %q that bai: %w", fname, err)
		}
	}

	log.Printf("da chay xong %d file migration", len(files))
	return nil
}
