# Checklist Capstone — Ticket Booking System

Tài liệu này trình bày những gì dự án **đã có**, đối chiếu trực tiếp với
từng mục bắt buộc trong `capstone-requirements.md` §4 (CKAD checklist) và
§3 (chuẩn microservices). Với mỗi mục: trạng thái + bằng chứng (đường dẫn
file thật trong repo, không phải mô tả suông).

Chú thích trạng thái: ✅ Hoàn thành · ⚠️ Một phần (còn khoảng trống có ghi
rõ) · ⏳ Chưa làm (có lý do/ưu tiên khác).

---

## 1. Tổng quan hệ thống

**Bài toán**: đặt vé xem phim — 5 service Go độc lập, mỗi service 1
database riêng, giao tiếp đồng bộ (HTTP qua Kubernetes DNS) và bất đồng bộ
(RabbitMQ).

| Service | Vai trò | Datastore |
|---|---|---|
| `api-gateway` | Cổng vào duy nhất (NodePort `30080` + Ingress), JWT, rate limit | — |
| `movie-service` | Phim, rạp, suất chiếu | Postgres riêng |
| `booking-service` | Giữ ghế (TTL), tạo booking | Postgres + Redis |
| `payment-service` | Xử lý thanh toán, publish sự kiện | Postgres + RabbitMQ (publisher) |
| `notification-service` | Nghe sự kiện, gửi email/QR | Postgres + RabbitMQ (consumer) + MailHog |

---

## 2. §4.1 — Application Design and Build

| # | Yêu cầu | Trạng thái | Bằng chứng |
|---|---|:---:|---|
| D1 | Custom image cho mỗi service | ✅ | 5 Dockerfile (`movie-booking/<svc>/Dockerfile`) + 2 sidecar image (`log-shipper`, `nginx-ambassador`); build qua `scripts/build.sh`, tag `1.0.0` |
| D2 | Deployment cho API; Job/CronJob cho batch | ✅ | `movie-booking/k8s/booking-service/cronjob.yaml` — `booking-cleanup-cronjob` chạy mỗi 5 phút, dùng đúng logic nghiệp vụ thật (`-cleanup-expired-holds`), verify sống bằng job thủ công (Completed, log đúng). Migration Job cũng có bản demo ở `labs/lab-1.3` |
| D3 | Multi-container: init và/hoặc sidecar | ✅ | Mỗi Pod (trừ RabbitMQ/datastore) có **cả init lẫn 2 sidecar**: `init-db-migrate` + `log-shipper` + `nginx-ambassador` |
| D4 | `emptyDir` chia sẻ giữa các container | ✅ | Volume `app-logs` (emptyDir) chia sẻ giữa `app` và `log-shipper` trong mọi Deployment |
| D5 | PVC, dữ liệu sống sót qua Pod delete/recreate | ✅ | 4 Postgres StatefulSet (`postgres-movie/booking/payment/notification`) + RabbitMQ PVC — đã verify sống việc dữ liệu còn nguyên qua Lab 4.4 |
| D6 | Label dùng cho selection/rollout/blue-green | ⚠️ | Label `app` nhất quán toàn bộ; blue/green label chỉ có trong `labs/lab-2.2`, chưa chuẩn hoá vào app chính |

## 3. §4.2 — Application Deployment

| # | Yêu cầu | Trạng thái | Bằng chứng |
|---|---|:---:|---|
| P1 | Mọi service chạy Deployment ≥1 replica | ✅ | 5 Deployment, `kubectl get pods -n movie-booking` → 12/12 Ready |
| P2 | Rolling update có tài liệu (`set image`/`apply` + `rollout status`) | ✅ | `labs/lab-2.1/run.sh` demo trực tiếp + `docs/demo-script.md` |
| P3 | Blue/green hoặc canary | ⚠️ | `labs/lab-2.2` (selector flip) — mới ở dạng demo lab, chưa đưa vào manifest chính |
| P4 | HPA trên ≥1 Deployment | ✅ | `movie-booking/k8s/autoscaling/booking-service-hpa.yaml` (target CPU) |
| P5 | Kustomize base + ≥1 overlay | ⚠️ | `movie-booking/k8s-kustomize/base` + `overlays/{dev,prod}` — chỉ bao `movie-service`, 4 service còn lại chưa có |
| P6 | Helm chart cài được, có upgrade + rollback | ✅ | `helm/movie-service` — 1 chart generic dùng chung cho cả 5 service (`values.yaml` có `services.<tên>` + `serviceName` chọn service), verify sống install → upgrade → rollback qua `labs/lab-5.4` |

## 4. §4.3 — Environment, Configuration & Security

| # | Yêu cầu | Trạng thái | Bằng chứng |
|---|---|:---:|---|
| C1 | ConfigMap inject qua env/volume | ✅ | Mỗi service có ConfigMap riêng (`envFrom`) |
| C2 | Secret cho credential | ⚠️ | `scripts/create-secrets.sh` / `generate-local-secrets.sh` sinh Secret runtime; mật khẩu training `postgres123` vẫn còn trong lịch sử git công khai — chỉ đổi mật khẩu thật khi cần, không rewrite history (quyết định có chủ đích) |
| C3 | SecurityContext: non-root, no priv-escalation, drop ALL, readOnly rootfs | ✅ | **Cả 5 service** (`app` container): `allowPrivilegeEscalation: false`, `capabilities.drop: [ALL]`, `readOnlyRootFilesystem: true` — trước đây chỉ `api-gateway` có, giờ đồng bộ toàn bộ, verify sống (12/12 Ready, smoke-test 22/22 PASS) |
| C4 | ServiceAccount + Role + RoleBinding, có Pod dùng | ✅ | `booking-observer` SA (get/list Pods) — gắn vào `booking-service` Deployment |
| C5 | ResourceQuota + LimitRange | ✅ | `movie-booking/k8s/quota/` |
| C6 | Mọi container có requests/limits | ✅ | Đã rà toàn bộ (app/init/sidecar + 4 Postgres + Redis + RabbitMQ + MailHog) — đều có |

## 5. §4.4 — Services and Networking

| # | Yêu cầu | Trạng thái | Bằng chứng |
|---|---|:---:|---|
| N1 | ClusterIP cho traffic nội bộ | ✅ | Mọi service backend chỉ expose ClusterIP |
| N2 | NodePort hoặc Ingress cho traffic ngoài | ✅ | Cả 2: NodePort `api-gateway` (30080) **và** Ingress |
| N3 | Ingress ≥2 path/host route khác backend | ✅ | Ingress route `/api` và `/movies` |
| N4 | NetworkPolicy: default-deny + allow tường minh + hạn chế egress | ✅* | `default-deny-ingress` phủ **toàn bộ** Pod trong namespace; 7 policy allow-list riêng cho từng datastore (mỗi Postgres chỉ nhận từ đúng service chủ, Redis chỉ từ booking-service, RabbitMQ chỉ từ payment+notification-service, MailHog chỉ từ notification-service) + egress restrict DNS-only ra ngoài namespace. *Docker Desktop không có CNI hỗ trợ policy nên không enforce được live — manifest đầy đủ, đã ghi rõ giới hạn |
| N5 | Endpoints đã verify, không Service mồ côi | ⚠️ | `movie-booking/k8s/phase5-testing/{smoke-test.sh,e2e-test.sh}` verify Endpoints — script còn giả định cứng context `docker-desktop` |

## 6. §4.5 — Observability and Maintenance

| # | Yêu cầu | Trạng thái | Bằng chứng |
|---|---|:---:|---|
| O1 | Liveness probe mọi Deployment dài hạn | ✅ | Cả 5 service + 4 Postgres + Redis + RabbitMQ |
| O2 | Readiness probe mọi Deployment dài hạn | ✅ | Như trên |
| O3 | Startup probe cho service khởi động chậm | ✅ | RabbitMQ có `startupProbe` (~150s), dùng `tcpSocket` thay vì `exec` để tránh bug CRI exec của Docker Desktop |
| O4 | README có mục debug (`logs`/`describe`/`events`/`top`) | ✅ | Mục "Debugging" trong `README.md` |
| O5 | API ổn định, không dùng API deprecated | ✅ | Toàn bộ dùng `apps/v1`, `networking.k8s.io/v1`, ... |

---

## 7. Pattern microservices đã áp dụng

| Pattern | Có thật hay chỉ demo | Ghi chú |
|---|:---:|---|
| Bounded context / single responsibility | Thật | movie / booking / payment / notification / gateway tách biệt |
| Database-per-service | Thật | 4 Postgres StatefulSet riêng biệt |
| API Gateway | Thật | JWT + rate limit + NodePort |
| Sync HTTP (Kubernetes DNS) | Thật | gateway→API, booking→movie, payment→booking |
| Async messaging (RabbitMQ topic exchange) | Thật | Code publish/consume thật, không phải chuỗi giả lập |
| Idempotent consumer | Thật | `INSERT … ON CONFLICT DO NOTHING` + ack sau khi xử lý |
| Polyglot persistence | Thật | Postgres + Redis (TTL) + RabbitMQ + MailHog |
| Không dùng chung 1 database | Thật | Tránh anti-pattern "shared database" |

## 8. Pattern Kubernetes/cloud-native đã áp dụng

| Pattern | Vị trí |
|---|---|
| Init container | `init-db-migrate` mọi service |
| Sidecar log shipper | `log-shipper` + `emptyDir` |
| Ambassador/proxy sidecar | `nginx-ambassador` (cổng 9090) |
| StatefulSet + PVC | 4 Postgres + RabbitMQ PVC |
| HPA | `booking-service-hpa` |
| Blue/green | `labs/lab-2.2` (demo, chưa vào main) |
| Job / CronJob | `booking-cleanup-cronjob` (production) + `labs/lab-1.3` (demo) |
| NetworkPolicy | Default-deny toàn namespace + 10 allow-list tường minh |
| RBAC | `booking-observer` SA/Role/RoleBinding |
| Kustomize | base + dev/prod overlay (chỉ `movie-service`) |
| Helm | 1 chart generic cho cả 5 service, chọn qua `serviceName` |

---

## 9. Giới hạn môi trường (không phải bug code)

- **Không có CNI hỗ trợ NetworkPolicy** trên Docker Desktop → NetworkPolicy apply được nhưng không enforce thật khi demo live.
- **Không có ingress controller cài sẵn** → Ingress apply được nhưng không có địa chỉ route thật; NodePort vẫn hoạt động bình thường để bù.
- **`kubectl exec`/exec-probe đôi khi lỗi** (bug CRI của Docker Desktop: `http: server gave HTTP response to HTTPS client`) → đã né bằng cách đổi probe RabbitMQ sang `tcpSocket` thay vì `exec`.

Cả 3 điểm trên đã ghi trong `README.md` mục "Known limitations" và
`docs/ckad-checklist.md`.

## 10. Đã verify sống (không chỉ đọc code)

- `bash scripts/verify-kubernetes.sh` → 12/12 Pod Ready, 5/5 Deployment available.
- `bash movie-booking/k8s/phase5-testing/smoke-test.sh` → **22/22 PASS**.
- `curl http://localhost:30080/healthz` → `200`.
- Helm install → upgrade → rollback (`labs/lab-5.4`) → xác nhận rollback đúng về image cũ.
- CronJob dọn hold hết hạn → chạy job thủ công → `Completed`, log đúng.
