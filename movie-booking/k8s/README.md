# k8s/ — Hạ tầng (Phase 3)

Postgres x4 (StatefulSet) + Redis (Deployment) + RabbitMQ (Deployment),
chưa bao gồm 5 microservice (đó là Phase 4).

## Áp dụng

```bash
cd movie-booking/k8s

# Bước 0 (nếu chưa có sẵn từ trước) - namespace
kubectl apply -f 00-namespace.yaml

# Validate cú pháp + schema TRƯỚC KHI apply thật (khuyến khích luôn làm bước này)
kubectl apply --dry-run=client -f postgres-movie/
kubectl apply --dry-run=client -f postgres-booking/
kubectl apply --dry-run=client -f postgres-payment/
kubectl apply --dry-run=client -f postgres-notification/
kubectl apply --dry-run=client -f redis/
kubectl apply --dry-run=client -f rabbitmq/

# Apply thật
kubectl apply -f postgres-movie/
kubectl apply -f postgres-booking/
kubectl apply -f postgres-payment/
kubectl apply -f postgres-notification/
kubectl apply -f redis/
kubectl apply -f rabbitmq/
```

## Theo dõi quá trình lên

```bash
kubectl get pods -n movie-booking -w
```

Postgres StatefulSet thường mất 10-20s để sẵn sàng (initdb). RabbitMQ mất
lâu hơn (~20-30s) — đây là lý do `readinessProbe` của nó có
`initialDelaySeconds: 20` thay vì 5 như Postgres/Redis.

## Verify từng thành phần

```bash
# Xem toàn bộ resource vừa tạo
kubectl get statefulset,deployment,pod,pvc,svc -n movie-booking

# PVC phải ở trạng thái Bound (không phải Pending)
kubectl get pvc -n movie-booking

# Exec vào 1 Postgres, kiểm tra kết nối được
kubectl exec -it postgres-movie-0 -n movie-booking -- psql -U postgres -d movie_service -c "\dt"
# (chưa có bảng nào - binh thường, vì migration chưa chạy, đó là việc của
# initContainer ở Phase 4)

# Kiểm tra DNS headless service resolve đúng Pod
kubectl run -it --rm debug --image=busybox --restart=Never -n movie-booking -- \
  nslookup postgres-movie.movie-booking.svc.cluster.local

# Redis
kubectl exec -it deploy/redis -n movie-booking -- redis-cli ping
# phải trả về "PONG"

# RabbitMQ - port-forward để xem Web UI
kubectl port-forward svc/rabbitmq 15672:15672 -n movie-booking
# rồi mở http://localhost:15672 (guest/guest)
```

## Troubleshooting

| Triệu chứng | Kiểm tra |
|---|---|
| Pod ở trạng thái `Pending` mãi | `kubectl describe pod <tên-pod> -n movie-booking` — thường do PVC không bind được (storageClass không tồn tại) |
| Postgres Pod `CrashLoopBackOff` | `kubectl logs postgres-movie-0 -n movie-booking` — kiểm tra lỗi initdb (nếu là lỗi "not empty", đúng vấn đề PGDATA subfolder đã fix sẵn trong manifest) |
| RabbitMQ Pod restart liên tục lúc mới lên | Bình thường trong ~20-30s đầu (đang khởi tạo Erlang VM) — chỉ lo nếu sau 1 phút vẫn chưa `Running` |
| `nslookup` không resolve được | Kiểm tra tên Service headless khớp CHÍNH XÁC với `serviceName` trong StatefulSet |

## Dọn dẹp (nếu cần làm lại từ đầu)

```bash
kubectl delete -f postgres-movie/ -f postgres-booking/ -f postgres-payment/ -f postgres-notification/ -f redis/ -f rabbitmq/

# PVC KHÔNG bị xoá tự động cùng StatefulSet (đây là thiết kế cố ý của k8s -
# bảo vệ dữ liệu khỏi bị xoá nhầm khi chỉ định xoá workload) - xoá riêng nếu
# thực sự muốn mất sạch data:
kubectl delete pvc -n movie-booking --all
```