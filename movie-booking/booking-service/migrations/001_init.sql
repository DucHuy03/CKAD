-- migrations/001_init.sql
--
-- Bảng bookings lưu LỊCH SỬ đặt vé (kể cả các lượt hold đã hết hạn/bị huỷ),
-- dùng cho việc tra cứu và báo cáo. Trạng thái "đang giữ ghế trong bao lâu"
-- (phần cần tự động hết hạn) thì KHÔNG nằm ở đây mà nằm ở Redis (key TTL) -
-- lý do: Postgres không có cơ chế tự xoá 1 dòng sau N giây, Redis thì có
-- sẵn (EXPIRE) và cực nhanh cho việc check "ghế này có đang bị giữ không".
--
-- status có 4 giá trị:
--   HOLD      - vừa giữ ghế, chờ thanh toán, có expires_at
--   CONFIRMED - thanh toán thành công (do payment-service gọi callback)
--   CANCELLED - user chủ động huỷ trước khi hết hạn
--   EXPIRED   - hold quá thời gian mà không thanh toán (CronJob Lab 1.3 sẽ set)

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS bookings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    showtime_id UUID NOT NULL,
    user_id     TEXT NOT NULL,
    seats       TEXT[] NOT NULL,
    total_price NUMERIC(10, 2) NOT NULL,
    status      TEXT NOT NULL DEFAULT 'HOLD'
                CHECK (status IN ('HOLD', 'CONFIRMED', 'CANCELLED', 'EXPIRED')),
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index phục vụ đúng 2 truy vấn hay dùng nhất:
-- 1. CronJob quét "HOLD nào đã quá expires_at" -> cần index theo (status, expires_at)
-- 2. Tra cứu lịch sử đặt vé theo suất chiếu -> index theo showtime_id
CREATE INDEX IF NOT EXISTS idx_bookings_status_expires ON bookings (status, expires_at);
CREATE INDEX IF NOT EXISTS idx_bookings_showtime_id ON bookings (showtime_id);