-- migrations/001_init.sql
--
-- Bảng notifications lưu lịch sử gửi email + vé QR. Giống payment-service,
-- không có FK sang booking_id/payment_id vì thuộc database của service khác.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS notifications (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id     TEXT NOT NULL,
    booking_id     TEXT NOT NULL,
    showtime_id    TEXT NOT NULL,
    user_id        TEXT NOT NULL,
    email          TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'PENDING'
                   CHECK (status IN ('PENDING', 'SENT', 'FAILED')),
    failure_reason TEXT NOT NULL DEFAULT '',
    qr_data_base64 TEXT NOT NULL DEFAULT '', -- luu lai anh QR (PNG, base64) de tra cuu lai qua GET API
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Idempotency: 1 payment_id chi nen tao ra 1 notification (tranh gui trung
-- email neu RabbitMQ redeliver message do consumer crash giua chung va
-- message chua duoc ack). UNIQUE constraint bien viec "check da xu ly
-- payment_id nay chua" thanh 1 INSERT ... ON CONFLICT DO NOTHING duy nhat,
-- khong can code phai tu SELECT roi moi INSERT (tranh race condition giua
-- 2 buoc do neu sau nay chay nhieu replica notification-service).
CREATE UNIQUE INDEX IF NOT EXISTS idx_notifications_payment_id_unique ON notifications (payment_id);