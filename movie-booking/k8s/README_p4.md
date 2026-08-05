# Phase 4 — Deploy 5 microservice lên k8s (4-container/Pod)

## Bước 1 — Build 7 image (5 service + 2 image dùng chung)

Docker Desktop k8s dùng CHUNG Docker daemon với lệnh `docker build` bạn gõ
tay — nghĩa là **không cần push lên registry nào**, build xong là k8s node
thấy ngay.

```bash
cd movie-booking

docker build -t movie-service:latest        ./movie-service
docker build -t booking-service:latest      ./booking-service
docker build -t payment-service:latest      ./payment-service
docker build -t notification-service:latest ./notification-service
docker build -t api-gateway:latest          ./api-gateway

docker build -t log-shipper:latest          ./shared-images/log-shipper
docker build -t nginx-ambassador:latest     ./shared-images/nginx-ambassador

docker images | grep -E "movie-service|booking-service|payment-service|notification-service|api-gateway|log-shipper|nginx-ambassador"
```

**Nếu quên bước này**, Pod sẽ bị `ImagePullBackOff` vì k8s cố pull từ Docker
Hub (không tồn tại) thay vì dùng image local.

## Bước 2 — Đảm bảo hạ tầng Phase 3 đã chạy (bao gồm MailHog vừa bổ sung)

```bash
kubectl apply -f k8s/mailhog/
kubectl get pods -n movie-booking
```

Phải thấy `postgres-movie-0`, `postgres-booking-0`, `postgres-payment-0`,
`postgres-notification-0`, `redis-...`, `rabbitmq-...`, `mailhog-...` đều
`Running`. Nếu chưa, quay lại Phase 3.

## Bước 3 — Apply theo ĐÚNG THỨ TỰ (quan trọng!)

`api-gateway` có initContainer chờ 4 service kia sống — apply nó SAU CÙNG:

```bash
kubectl apply -f k8s/movie-service/
kubectl apply -f k8s/booking-service/
kubectl apply -f k8s/payment-service/
kubectl apply -f k8s/notification-service/

# Cho 4 cai tren Running het roi moi apply gateway
kubectl get pods -n movie-booking -w
# (Ctrl+C khi thay 4 Pod deu 3/3 Running... cho gateway)

kubectl apply -f k8s/api-gateway/
```

## Bước 4 — Theo dõi quá trình lên, quan sát đúng pattern 4-container

```bash
kubectl get pods -n movie-booking -w
```

Mỗi Pod sẽ hiện `0/3 -> ... -> 3/3 Running` (3 container trong `containers:`,
initContainer không tính vào tỉ lệ này vì nó không phải container thường trực).

```bash
# Xem chi tiết 1 Pod, thay ro 4 container (1 init + 3 app)
kubectl describe pod -l app=movie-service -n movie-booking

# Xem log RIENG TUNG container - day chinh la Lab 1.2
kubectl logs -l app=movie-service -n movie-booking -c app
kubectl logs -l app=movie-service -n movie-booking -c log-shipper
kubectl logs -l app=movie-service -n movie-booking -c nginx-ambassador
kubectl logs -l app=movie-service -n movie-booking -c init-db-migrate
```

`log-shipper` và `app` phải in ra **nội dung giống nhau** (cùng đọc/ghi 1
file qua `emptyDir`) — đây là cách xác nhận trực quan pattern sidecar hoạt động đúng.

## Bước 5 — Test toàn bộ flow qua NodePort

```bash
curl http://localhost:30080/healthz

TOKEN=$(curl -s -X POST http://localhost:30080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user-001","password":"anything"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])")

curl http://localhost:30080/api/movies
# ... lap lai toan bo flow nhu da test o docker-compose (xem README tung service)
```

MailHog Web UI trên k8s: `kubectl port-forward svc/mailhog 8025:8025 -n movie-booking` rồi mở http://localhost:8025

RabbitMQ Web UI: `kubectl port-forward svc/rabbitmq 15672:15672 -n movie-booking`

## Troubleshooting

| Triệu chứng | Nguyên nhân thường gặp | Lệnh kiểm tra |
|---|---|---|
| `ImagePullBackOff` | Quên build image, hoặc quên `imagePullPolicy: IfNotPresent` | `kubectl describe pod <tên> -n movie-booking` xem `Events` |
| Pod kẹt ở `Init:0/1` | initContainer đang chờ DB/backend chưa sẵn sàng | `kubectl logs <tên-pod> -n movie-booking -c <tên-init-container>` |
| Pod `Running` nhưng `curl` qua Service timeout | Có thể ambassador container chưa `Ready` (readinessProbe fail) — cả 3 container phải Ready thì Pod mới nhận traffic | `kubectl describe pod <tên> -n movie-booking`, xem phần `Conditions` |
| `log-shipper` không in gì | File `app.log` chưa được tạo (app container lỗi, hoặc `LOG_FILE_PATH` giữa 2 container không khớp) | So sánh env `LOG_FILE_PATH` giữa `app` và `log-shipper` bằng `kubectl exec` |
| `api-gateway` Pod kẹt `Init:0/1` mãi | 1 trong 4 backend chưa `Running`, hoặc Service của backend đó chưa tồn tại | `kubectl logs <pod-api-gateway> -n movie-booking -c wait-backends` xem đang chờ service nào |

## Dọn dẹp

```bash
kubectl delete -f k8s/movie-service/ -f k8s/booking-service/ -f k8s/payment-service/ -f k8s/notification-service/ -f k8s/api-gateway/
```