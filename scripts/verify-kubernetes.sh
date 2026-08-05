#!/usr/bin/env bash
# Kiem tra nhanh sau deploy. Script chi doc cluster, khong sua resource nao.
# Neu co Pod chua Ready, no in describe va log cua container truoc de debug.

set -euo pipefail
NAMESPACE="${NAMESPACE:-movie-booking}"
# 10 giay khong du cho init container migrate va ket noi broker sau restart.
# Co the override, vi du VERIFY_TIMEOUT=30s bash scripts/verify-kubernetes.sh.
VERIFY_TIMEOUT="${VERIFY_TIMEOUT:-90s}"
FAILED=0

for deployment in movie-service booking-service payment-service notification-service api-gateway; do
  if ! kubectl rollout status "deployment/$deployment" -n "$NAMESPACE" --timeout="$VERIFY_TIMEOUT"; then
    printf '\n--- Diagnostic for %s ---\n' "$deployment" >&2
    kubectl describe deployment "$deployment" -n "$NAMESPACE" >&2 || true
    kubectl logs "deployment/$deployment" -n "$NAMESPACE" --all-containers --tail=80 >&2 || true
    FAILED=1
  fi
done

if [[ "$FAILED" -ne 0 ]]; then
  echo "Kubernetes verification failed." >&2
  exit 1
fi

echo "All five application deployments are available."
kubectl get pods,svc -n "$NAMESPACE"
