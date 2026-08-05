-- migrations/001_init.sql
--
-- Bảng payments lưu lịch sử xử lý thanh toán. KHÔNG có foreign key tới
-- booking_id/showtime_id vì chúng thuộc database của service khác
-- (booking-service, movie-service) - vi phạm nguyên tắc database-per-service
-- nếu tạo FK xuyên service. Việc booking_id có hợp lệ hay không được xác
-- nhận bằng cách gọi API sang booking-service lúc xử lý thanh toán, không
-- phải bằng ràng buộc ở tầng DB.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS payments (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id     UUID NOT NULL,
    showtime_id    UUID NOT NULL,
    user_id        TEXT NOT NULL,
    amount         NUMERIC(10, 2) NOT NULL CHECK (amount > 0),
    method         TEXT NOT NULL DEFAULT 'CARD',
    status         TEXT NOT NULL DEFAULT 'PENDING'
                   CHECK (status IN ('PENDING', 'SUCCESS', 'FAILED')),
    failure_reason TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 1 booking chỉ nên có (nhiều nhất 1 payment SUCCESS), nhưng có thể có
-- nhiều lần thử FAILED trước đó -> không đặt UNIQUE constraint tuyệt đối
-- trên booking_id, chỉ index để tra cứu nhanh lịch sử thanh toán của 1 booking.
CREATE INDEX IF NOT EXISTS idx_payments_booking_id ON payments (booking_id);