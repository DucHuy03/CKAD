# Lưu ý trước khi chạy các lab

Tài liệu này dùng cho môi trường **WSL + Docker Desktop Kubernetes**. Mỗi lab có hai lệnh:

```bash
# Chạy demo: có thể tạo, sửa hoặc xoá tài nguyên Kubernetes.
bash labs/lab-X.Y/run.sh

# Kiểm tra cấu trúc/evidence: không chủ đích thay đổi cluster.
bash labs/lab-X.Y/check.sh
```

## Chuẩn bị chung

Chạy tại thư mục gốc project trong WSL:

```bash
cd /mnt/d/Ticket-booking-system
kubectl config current-context       # Kỳ vọng: docker-desktop
kubectl get nodes
source .env.local                    # Nạp DB_PASSWORD, JWT_SECRET, RabbitMQ_PASSWORD...
```

Docker Desktop phải đang chạy và đã bật WSL Integration cho distribution đang dùng. Nếu chưa có `.env.local`, tạo một lần bằng `bash scripts/generate-local-secrets.sh "Ten Cua Ban" 0311`, sau đó chạy `source .env.local`.

> `DB_PASSWORD` chỉ đúng với Postgres PVC nếu đó là mật khẩu đã dùng khi PVC được khởi tạo. Nếu bạn tạo `.env.local` **sau** lần deploy đầu tiên, PVC cũ có thể vẫn dùng mật khẩu cũ. Khi đó không được reset PVC nếu chưa chấp nhận mất dữ liệu; hãy dùng lại mật khẩu cũ cho lab.

Các lab có truy cập database cần thêm biến sau (thay `MAT_KHAU_POSTGRES_GOC` bằng mật khẩu đã khởi tạo Postgres PVC):

```bash
export LAB_DB_PASSWORD='MAT_KHAU_POSTGRES_GOC'
```

Chỉ dùng `export LAB_DB_PASSWORD="$DB_PASSWORD"` khi giá trị `DB_PASSWORD` hiện tại thực sự là mật khẩu của Postgres đang chạy.

## Giữ namespace `lab` sạch

Nhiều lab cùng dùng namespace `lab`; các Pod/Job cũ có thể làm đầy quota 10 Pod, khiến lab mới báo `exceeded quota`. Trước khi đổi sang lab khác, có thể dọn **chỉ namespace demo `lab`**:

```bash
kubectl delete cronjob --all -n lab
kubectl delete all --all -n lab
```

Hai lệnh này xoá tài nguyên demo trong `lab`, không xoá hệ thống chính ở namespace `movie-booking`. Không chạy chúng khi bạn còn cần giữ evidence của một lab để demo/chấm.

## Lưu ý theo từng lab

| Lab | Cần chuẩn bị / tác động cần biết |
| --- | --- |
| 1.1 | Cần `LAB_DB_PASSWORD`. Script tạo `lab-db-credentials`, tạo Debug Pod ở namespace `lab` và port-forward tạm thời. |
| 1.2 | Cần `LAB_DB_PASSWORD`. Init container chạy migration; nếu Pod đứng `PodInitializing`, xem `kubectl logs movie-catalog-multi -c init-db-migrate -n lab`. Lỗi `28P01` nghĩa là sai mật khẩu Postgres PVC. |
| 1.3 | Tạo Job và CronJob; sau khi demo nên dọn CronJob/Job để tránh quota đầy: `kubectl delete cronjob,job --all -n lab`. Không cần `export` bí mật. |
| 1.4 | Dùng các image local của project và tạo Pod phục vụ chẩn đoán ở `lab`. Có thể chọn tag bằng `export IMAGE_TAG=1.0.0`. Dọn `lab` trước nếu quota đã đầy. |
| 2.1 | Cần Docker và `movie-service` đang chạy trong `movie-booking`. Lab **cố ý** deploy image không tồn tại để minh hoạ rollback. Script hiện tự khôi phục image và `APP_VERSION` ban đầu, kể cả khi dừng bằng Ctrl+C. |
| 2.2 | Dùng namespace `lab`, thực hiện blue/green và port-forward. Không chạy song song với lab khác dùng cùng tên service/Pod. |
| 2.3 | Cần Internet, quyền cài metrics-server ở `kube-system`, và có thể thay đổi số replica của `booking-service`. Không chạy trên cluster production. |
| 2.4 | Cần `kubectl kustomize`, image `movie-service:v1` và `movie-service:v2` (chạy Lab 2.1 trước để build). Cần Secret `movie-service-secret` trong namespace `lab` (chạy `export LAB_DB_PASSWORD='MAT_KHAU_POSTGRES_GOC'; bash scripts/create-lab-secrets.sh` một lần trước khi chạy lab này lần đầu). |
| 3.1 | Cần `LAB_DB_PASSWORD`; lab tạo Secret/ConfigMap mẫu ở namespace `lab`. |
| 3.2 | Không cần `export` bí mật; kiểm tra security context và resource constraints. |
| 3.3 | Dùng các manifest RBAC trong `movie-booking/lab-3.3`; cần quyền tạo ServiceAccount, Role và RoleBinding ở `lab`. |
| 3.4 | Tạo ResourceQuota/LimitRange cho `lab`. Chạy lab này có thể làm các lab khác không tạo được Pod nếu tài nguyên cũ chưa được dọn. |
| 4.1–4.4 | Các lab networking dùng namespace `lab`; nên dọn namespace trước và chạy lần lượt để NetworkPolicy/Service cũ không ảnh hưởng nhau. |
| 5.1–5.3 | Kiểm tra/triển khai workload cơ bản trong `lab`; áp dụng quy tắc dọn namespace và không chạy song song. Lab 5.2 có thể đổi namespace bằng `export NAMESPACE=lab`. |
| 5.4 | Cần cài Helm (`helm version`) và các image local đã build. Có thể đặt `export RELEASE=lab54-movie-service` và `export NAMESPACE=lab` trước khi chạy. |

## Trình tự demo gợi ý

1. Deploy hệ thống chính một lần: `bash scripts/deploy-kubernetes.sh` rồi `bash scripts/verify-kubernetes.sh`.
2. Nạp `.env.local`, đặt `LAB_DB_PASSWORD` đúng với Postgres PVC.
3. Với mỗi lab: dọn `lab` nếu không cần giữ evidence cũ, chạy `run.sh`, trình bày kết quả, rồi chạy `check.sh`.
4. Các lab có phụ thuộc: chạy **2.1 trước 2.4**; chạy **3.4 sau cùng** trong nhóm lab dùng namespace `lab` hoặc dọn namespace trước mỗi lab.
