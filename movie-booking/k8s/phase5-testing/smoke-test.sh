#!/bin/bash
# smoke-test.sh - kiem tra CAU TRUC ha tang truoc khi test nghiep vu.
#
# Khac voi e2e-test.sh (kiem tra LUONG NGHIEP VU dung/sai), script nay chi
# tra loi cau hoi: "cluster co dang o trang thai KHOE MANH de bat dau test
# khong" - chay script nay TRUOC e2e-test.sh, tranh mat cong debug nghiep vu
# trong khi nguyen nhan that su chi la 1 Pod chua len xong.

set -uo pipefail

NAMESPACE="movie-booking"
# Override with: EXPECTED_CONTEXT=minikube bash smoke-test.sh
EXPECTED_CONTEXT="${EXPECTED_CONTEXT:-docker-desktop}"
PASS=0
FAIL=0

green() { printf "\033[32m%s\033[0m\n" "$1"; }
red()   { printf "\033[31m%s\033[0m\n" "$1"; }

check() {
  local desc="$1"
  local result="$2"
  if [ "$result" = "0" ]; then
    green "  [PASS] $desc"
    PASS=$((PASS+1))
  else
    red "  [FAIL] $desc"
    FAIL=$((FAIL+1))
  fi
}

echo "=== 1. Kiem tra context kubectl ==="
CTX=$(kubectl config current-context 2>/dev/null)
echo "  context hien tai: $CTX (mong doi: $EXPECTED_CONTEXT)"
[ "$CTX" = "$EXPECTED_CONTEXT" ]
check "context la $EXPECTED_CONTEXT (doi bang bien EXPECTED_CONTEXT neu dung cluster khac)" $?

echo ""
echo "=== 2. Kiem tra hạ tầng (Postgres x4, Redis, RabbitMQ, MailHog) ==="
for name in postgres-movie-0 postgres-booking-0 postgres-payment-0 postgres-notification-0; do
  phase=$(kubectl get pod "$name" -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null)
  [ "$phase" = "Running" ]
  check "$name dang Running" $?
done

for deploy in redis rabbitmq mailhog; do
  ready=$(kubectl get deploy "$deploy" -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null)
  [ "${ready:-0}" -ge 1 ] 2>/dev/null
  check "$deploy co it nhat 1 replica Ready" $?
done

echo ""
echo "=== 3. Kiem tra 5 microservice - PHAI dung 3/3 container (init khong tinh) ==="
for svc in movie-service booking-service payment-service notification-service api-gateway; do
  ready_str=$(kubectl get pods -n "$NAMESPACE" -l app="$svc" -o jsonpath='{.items[0].status.containerStatuses[*].ready}' 2>/dev/null)
  ready_count=$(echo "$ready_str" | tr ' ' '\n' | grep -c "^true$")
  [ "$ready_count" = "3" ]
  check "$svc co dung 3/3 container Ready (app+log-shipper+nginx-ambassador)" $?
done

echo ""
echo "=== 4. Kiem tra Service co Endpoints (khong bi selector mismatch) ==="
for svc in movie-service booking-service payment-service notification-service api-gateway redis rabbitmq mailhog; do
  ep_count=$(kubectl get endpoints "$svc" -n "$NAMESPACE" -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null | wc -w)
  [ "$ep_count" -ge 1 ] 2>/dev/null
  check "$svc co it nhat 1 endpoint" $?
done

echo ""
echo "=== 5. Xac nhan pattern sidecar: log cua 'app' va 'log-shipper' PHAI khop nhau ==="
POD=$(kubectl get pods -n "$NAMESPACE" -l app=movie-service -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$POD" ]; then
  APP_LAST_LINE=$(kubectl logs "$POD" -n "$NAMESPACE" -c app --tail=1 2>/dev/null)
  SHIPPER_LAST_LINE=$(kubectl logs "$POD" -n "$NAMESPACE" -c log-shipper --tail=1 2>/dev/null)
  [ -n "$APP_LAST_LINE" ] && [ "$APP_LAST_LINE" = "$SHIPPER_LAST_LINE" ]
  check "dong log cuoi cung cua app va log-shipper GIONG NHAU (pod: $POD)" $?
  echo "    app         : $APP_LAST_LINE"
  echo "    log-shipper : $SHIPPER_LAST_LINE"
else
  check "tim thay Pod movie-service de kiem tra log" 1
fi

echo ""
echo "================================"
green "PASS: $PASS"
[ "$FAIL" -gt 0 ] && red "FAIL: $FAIL" || echo "FAIL: $FAIL"
echo "================================"

if [ "$FAIL" -gt 0 ]; then
  echo ""
  echo "Con loi -> chay 'kubectl get pods -n $NAMESPACE' va 'kubectl describe pod <ten> -n $NAMESPACE' de xem chi tiet TRUOC KHI chay e2e-test.sh"
  exit 1
fi

echo ""
green "Ha tang khoe manh - san sang chay e2e-test.sh"