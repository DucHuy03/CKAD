# Ticket Booking Service — CKAD Capstone

Ticket Booking Service is a Kubernetes-focused cinema booking system. The
application source is deliberately kept in `movie-booking/` during the first
reorganisation phase so existing Docker Compose and Kubernetes commands remain
valid. New documentation and automation live at the repository root.

## Services

| Service | Responsibility |
| --- | --- |
| api-gateway | Public entry point, JWT and rate limiting |
| movie-service | Movies, cinemas and showtimes |
| booking-service | Seat holds and bookings; uses Redis |
| payment-service | Payment processing and payment events |
| notification-service | Consumes events and sends notifications |

## Repository map

- `movie-booking/`: current application source, Docker Compose and Kubernetes manifests.
- `docs/`: architecture, CKAD evidence and the demo/runbook documentation.
- `labs/`: one stable entry point per CKAD lab. A lab check reports either PASS,
  FAIL, or TODO; it never mutates the cluster.
- `scripts/`: reusable build/deploy/check helpers. Scripts locate the repository
  root themselves, so they can be executed from any working directory.

## Phase 1 status

Phase 1 adds documentation and repeatable checks, and fixes broken relative
paths in existing lab scripts. It intentionally does not move live source or
manifests yet. That migration happens only after the new paths have been
validated in Phase 2.

Run the static capstone audit with:

```bash
bash scripts/check-capstone.sh
```

Before deploying Kubernetes manifests, create runtime Secrets from values that
are not stored in Git:

```bash
export DB_PASSWORD='...'
export RABBITMQ_PASSWORD='...'
export RABBITMQ_USERNAME='...'
export JWT_SECRET='...'
bash scripts/create-secrets.sh
```

Hoac tao mot bo secret local (co random suffix, file khong duoc commit) roi
nap vao shell hien tai:

```bash
bash scripts/generate-local-secrets.sh "Ten Cua Ban" 0311
source .env.local
bash scripts/deploy-kubernetes.sh
```

Lab manifests that need a database use an isolated Secret. Create it before
running Labs 1.1–3.2. `LAB_DB_PASSWORD` must match the `DB_PASSWORD` used
when deploying the application:

```bash
export LAB_DB_PASSWORD='...'
bash scripts/create-lab-secrets.sh
```

Run a lab check with:

```bash
bash labs/lab-1.1/check.sh
```

## Demo labs

Mỗi lab có một script demo và một script kiểm tra tại `labs/lab-x.y/`:

```bash
# Demo Lab 1.1 (có tạo/xóa resource tùy nội dung lab).
bash labs/lab-1.1/run.sh

# Kiểm tra evidence của Lab 1.1, an toàn khi chạy nhiều lần.
bash labs/lab-1.1/check.sh
```

Để chạy toàn bộ ứng dụng trước các lab cần service thật, export bốn secret
runtime rồi dùng `bash scripts/deploy-kubernetes.sh`.

### Chạy từ Windows PowerShell

`bash` trên máy Windows thường là WSL, vì vậy cần đặt biến môi trường theo cú
pháp PowerShell (không dùng `export`), rồi gọi script:

```powershell
$env:DB_PASSWORD = 'postgres-password-cua-ban'
$env:JWT_SECRET = 'jwt-secret-dai-va-ngau-nhien'
$env:RABBITMQ_USERNAME = 'appuser'
$env:RABBITMQ_PASSWORD = 'rabbitmq-password-cua-ban'

bash scripts/deploy-kubernetes.sh
bash scripts/verify-kubernetes.sh
```

See `docs/ckad-checklist.md` for the current gap list and `docs/architecture.md`
for service communication.
