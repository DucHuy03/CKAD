# Phase 5 — Test end-to-end trên cluster

## Chạy

```bash
cd movie-booking/k8s/phase5-testing
chmod +x smoke-test.sh e2e-test.sh

./smoke-test.sh   # kiem tra CAU TRUC (Pod/Service/pattern 4-container)
./e2e-test.sh     # kiem tra LUONG NGHIEP VU that (login -> dat ve -> thanh toan -> email)
```

Yêu cầu: đã cài `jq`, `kubectl` trỏ đúng context `docker-desktop`, đã hoàn
tất Phase 4 (5 Pod đều `3/3 Running`).

## Vì sao tách 2 script thay vì gộp làm 1

`smoke-test.sh` trả lời **"hạ tầng có khoẻ không"**, `e2e-test.sh` trả lời
**"nghiệp vụ có đúng không"**. Tách riêng để khi có lỗi, bạn biết ngay nên
tìm ở tầng nào trước — chạy `e2e-test.sh` khi hạ tầng chưa ổn sẽ ra hàng loạt
lỗi khó hiểu (timeout, connection refused) che mất nguyên nhân gốc thật sự.

## `smoke-test.sh` kiểm tra gì

1. Context kubectl đúng `docker-desktop` (tránh táo tợn chạy nhầm cluster khác)
2. 4 Postgres StatefulSet Pod đều `Running`
3. Redis/RabbitMQ/MailHog Deployment có ít nhất 1 replica Ready
4. **5 microservice đều đúng 3/3 container Ready** (app + log-shipper + nginx-ambassador — initContainer không tính vào con số này)
5. Mọi Service đều có Endpoints (bắt được lỗi selector mismatch sớm — chính là nội dung Lab 4.1 sau này)
6. **Dòng log cuối cùng của container `app` và `log-shipper` phải giống hệt nhau** — xác nhận trực quan bằng script (không chỉ nhìn bằng mắt) rằng pattern sidecar hoạt động đúng

## `e2e-test.sh` kiểm tra gì

Chạy lại đúng flow đã test thủ công ở docker-compose (Phase 1), nhưng:
- Gọi qua `http://localhost:30080` (NodePort thật của k8s)
- Test cả **route công khai** (xem phim không cần token) lẫn **route bảo vệ** (tạo phim/đặt vé cần token, và assert rõ ràng `401` khi thiếu token — không chỉ test đường happy path)
- Retry có chủ đích ở bước thanh toán (giả lập ngẫu nhiên ~90% thành công) và bước chờ notification (bất đồng bộ qua RabbitMQ, cần thời gian xử lý)
- Dùng tên ghế ngẫu nhiên (`$RANDOM`) mỗi lần chạy — tránh xung đột "ghế đã được đặt" khi chạy script nhiều lần liên tiếp

## Khi có FAIL

- `smoke-test.sh` FAIL → quay lại `kubectl describe pod <tên>` / `kubectl logs` theo đúng Pod bị báo lỗi, xem `PHASE4-README.md` mục Troubleshooting
- `e2e-test.sh` FAIL ở bước cụ thể → response JSON đầy đủ được in ra ngay phía trên dòng `[FAIL]`, đọc `error` field trong đó trước khi hỏi thêm

## Sau khi cả 2 script PASS 100%

Đây là mốc hoàn tất **Phase 5** — hệ thống chạy đúng trên k8s y hệt lúc chạy
trên docker-compose, nhưng đúng kiến trúc 4-container/Pod theo yêu cầu ban
đầu. Từ đây có thể tự tin **bắt đầu Phase 6 (20 lab CKAD)** vì mọi thứ lab
sẽ "phá" (đổi label, sửa selector sai có chủ đích, scale, rolling update...)
đều có 1 baseline đã biết chắc là đúng để so sánh/khôi phục lại.