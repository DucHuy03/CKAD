-- migrations/001_init.sql
--
-- Migration đầu tiên: tạo 3 bảng cốt lõi cho movie-service.
-- Dùng "IF NOT EXISTS" để migration idempotent (chạy lại nhiều lần không lỗi),
-- vì initContainer có thể bị restart và chạy lại migration nhiều lần trong đời Pod.

-- Bật extension sinh UUID, dùng làm primary key thay vì auto-increment int
-- (UUID tránh lộ số lượng record, và dễ merge dữ liệu giữa các môi trường/service).
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS movies (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title            TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    duration_minutes INTEGER NOT NULL CHECK (duration_minutes > 0),
    genre            TEXT NOT NULL DEFAULT '',
    release_date     DATE,
    poster_url       TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS cinemas (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    address    TEXT NOT NULL DEFAULT '',
    city       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- showtimes tham chiếu movies + cinemas.
-- ON DELETE CASCADE: xoá phim/rạp thì các suất chiếu liên quan cũng bị xoá theo,
-- tránh showtime "mồ côi" trỏ tới movie_id không còn tồn tại.
CREATE TABLE IF NOT EXISTS showtimes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    movie_id     UUID NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    cinema_id    UUID NOT NULL REFERENCES cinemas(id) ON DELETE CASCADE,
    room_name    TEXT NOT NULL,
    start_time   TIMESTAMPTZ NOT NULL,
    end_time     TIMESTAMPTZ NOT NULL,
    price        NUMERIC(10, 2) NOT NULL CHECK (price >= 0),
    total_seats  INTEGER NOT NULL CHECK (total_seats > 0),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index cho các query hay dùng: booking-service tra showtime theo movie hoặc theo cinema.
CREATE INDEX IF NOT EXISTS idx_showtimes_movie_id  ON showtimes (movie_id);
CREATE INDEX IF NOT EXISTS idx_showtimes_cinema_id ON showtimes (cinema_id);
CREATE INDEX IF NOT EXISTS idx_showtimes_start_time ON showtimes (start_time);