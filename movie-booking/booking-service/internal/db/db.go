// Package db - giống hệt pattern ở movie-service (Connect / WaitForDB /
// RunMigrations). Xem comment chi tiết ở movie-service/internal/db/db.go,
// ở đây không lặp lại để tránh dài dòng.
package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "github.com/lib/pq"

	"booking-service/internal/config"
)

func Connect(cfg config.Config) (*sql.DB, error) {
	dbConn, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("mo ket noi postgres that bai: %w", err)
	}
	dbConn.SetMaxOpenConns(10)
	dbConn.SetMaxIdleConns(5)
	dbConn.SetConnMaxLifetime(5 * time.Minute)
	return dbConn, nil
}

func WaitForDB(dbConn *sql.DB, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		lastErr = dbConn.Ping()
		if lastErr == nil {
			return nil
		}
		log.Printf("cho postgres san sang... (%v)", lastErr)
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("het thoi gian cho postgres sau %s: %w", timeout, lastErr)
}

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
	sort.Strings(files)

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
