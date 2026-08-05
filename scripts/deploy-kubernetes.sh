#!/usr/bin/env bash
# Deploy day du Ticket Booking len Kubernetes cua Docker Desktop.
# Script KHONG luu mat khau trong repo: hay export DB_PASSWORD, JWT_SECRET,
# RABBITMQ_USERNAME va RABBITMQ_PASSWORD truoc khi chay.
# Co the doi IMAGE_TAG=1.0.1 neu ban muon phat hanh mot phien ban image moi.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
K8S_ROOT="$PROJECT_ROOT/movie-booking/k8s"
NAMESPACE="${NAMESPACE:-movie-booking}"
IMAGE_TAG="${IMAGE_TAG:-1.0.0}"

# Dung chung mot ham de thong bao tung moc deploy tren terminal.
step() {
  printf '\n==> %s\n' "$1"
}

# Kiem tra som de loi thieu cong cu/secret ro rang hon loi Pod kho doc.
command -v docker >/dev/null || { echo "Thieu docker" >&2; exit 1; }
command -v kubectl >/dev/null || { echo "Thieu kubectl" >&2; exit 1; }
: "${DB_PASSWORD:?Set DB_PASSWORD before deployment}"
: "${JWT_SECRET:?Set JWT_SECRET before deployment}"
: "${RABBITMQ_USERNAME:?Set RABBITMQ_USERNAME before deployment}"
: "${RABBITMQ_PASSWORD:?Set RABBITMQ_PASSWORD before deployment}"

step "Build local images (tag: $IMAGE_TAG)"
# Goi ro bang bash thay vi chay truc tiep file .sh. Cach nay hoat dong ca khi
# repo nam tren o D: cua Windows/WSL va file chua co executable bit.
IMAGE_TAG="$IMAGE_TAG" bash "$SCRIPT_DIR/build.sh"

step "Create namespace, policy and runtime secrets"
kubectl apply -f "$K8S_ROOT/00-namespace.yaml"
kubectl apply -f "$K8S_ROOT/quota/"
kubectl apply -f "$K8S_ROOT/security/"
NAMESPACE="$NAMESPACE" bash "$SCRIPT_DIR/create-secrets.sh"

step "Deploy stateful dependencies"
for component in postgres-movie postgres-booking postgres-payment postgres-notification redis rabbitmq mailhog; do
  kubectl apply -f "$K8S_ROOT/$component/"
done
kubectl rollout status statefulset/postgres-movie -n "$NAMESPACE" --timeout=180s
kubectl rollout status statefulset/postgres-booking -n "$NAMESPACE" --timeout=180s
kubectl rollout status statefulset/postgres-payment -n "$NAMESPACE" --timeout=180s
kubectl rollout status statefulset/postgres-notification -n "$NAMESPACE" --timeout=180s
kubectl rollout status deployment/redis -n "$NAMESPACE" --timeout=180s
kubectl rollout status deployment/rabbitmq -n "$NAMESPACE" --timeout=180s

# RabbitMQ luu user/password trong PVC luc khoi tao. Dong bo bang rabbitmqctl
# can kubectl exec, nhung Docker Desktop co the tam thoi khong ho tro exec.
# Vi vay mac dinh BO QUA buoc nay: lan deploy moi sau khi reset PVC da tu doc
# Secret dung. Chi bat khi can sua credential trong PVC CU va exec dang hoat
# dong: RECONCILE_RABBITMQ_CREDENTIALS=true bash scripts/deploy-kubernetes.sh.
if [[ "${RECONCILE_RABBITMQ_CREDENTIALS:-false}" == "true" ]]; then
  if ! NAMESPACE="$NAMESPACE" timeout 30s bash "$SCRIPT_DIR/reconcile-rabbitmq-credentials.sh"; then
    cat >&2 <<'EOF'
RabbitMQ credential reconciliation failed.
The existing RabbitMQ PVC may contain an older password, or kubectl exec is unavailable.
Use the original RabbitMQ credentials, or reset the local rabbitmq-data PVC only if losing queued messages is acceptable.
EOF
    exit 1
  fi
fi

step "Deploy application services in dependency order"
for component in movie-service booking-service payment-service notification-service; do
  kubectl apply -f "$K8S_ROOT/$component/"
  kubectl rollout status "deployment/$component" -n "$NAMESPACE" --timeout=180s
done
kubectl apply -f "$K8S_ROOT/api-gateway/"
kubectl rollout status deployment/api-gateway -n "$NAMESPACE" --timeout=180s

# NetworkPolicy duoc bat sau khi rollout: tranh chan traffic khoi dong trong
# luc dang debug va van giu duoc bai lab network isolation sau khi thanh cong.
step "Apply network policy and autoscaling"
kubectl apply -f "$K8S_ROOT/network/"
kubectl apply -f "$K8S_ROOT/autoscaling/"

step "Deployment completed"
kubectl get pods,svc -n "$NAMESPACE"
echo "Open http://localhost:30080/healthz"
