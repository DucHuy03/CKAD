#!/usr/bin/env bash

# Stop on errors, unset variables, and failed commands in pipes.
set -euo pipefail

# Resolve the demo manifest from this script so it can be launched anywhere.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
: "${LAB_DB_PASSWORD:?Set LAB_DB_PASSWORD before running this lab}"

NAMESPACE="lab"
DEMO_STARTED=false

restore_original_state() {
    local exit_code=$?

    # Before the demo starts, nothing was created, so no cleanup is needed.
    if [[ "$DEMO_STARTED" != true ]]; then
        exit "$exit_code"
    fi

    echo
    echo "Cleaning up Lab 3.1 demo resources..."
    kubectl delete pod movie-service-flags-demo -n "$NAMESPACE" --ignore-not-found >/dev/null || true
    kubectl delete configmap movie-service-feature-flags -n "$NAMESPACE" --ignore-not-found >/dev/null || true
    kubectl delete secret movie-service-db-secret -n "$NAMESPACE" --ignore-not-found >/dev/null || true
    exit "$exit_code"
}
trap restore_original_state EXIT INT TERM

echo "========================================"
echo " Lab 3.1 - ConfigMap & Secret Injection"
echo "========================================"

# From here, the script creates cluster resources; the trap must clean them up.
DEMO_STARTED=true

echo
echo "[1/6] Creating Secret..."

kubectl create secret generic movie-service-db-secret -n "$NAMESPACE" \
    --from-literal=DB_PASSWORD="$LAB_DB_PASSWORD" \
    --dry-run=client -o yaml | kubectl apply -f -

echo
echo "--- Secret Verify ---"

kubectl get secret movie-service-db-secret -n "$NAMESPACE" \
    -o jsonpath='{.data.DB_PASSWORD}' | base64 -d

echo

echo
echo "[2/6] Creating ConfigMap..."

kubectl create configmap movie-service-feature-flags -n "$NAMESPACE" \
    --from-literal=ENABLE_PROMO_BANNER=true \
    --from-literal=MAX_HOLD_SEATS_PER_USER=6 \
    --from-literal=MAINTENANCE_MODE=false \
    --dry-run=client -o yaml | kubectl apply -f -

echo
kubectl get configmap movie-service-feature-flags -n "$NAMESPACE" -o yaml

echo
echo "[3/6] Applying demo Pod..."

kubectl apply -n "$NAMESPACE" -f "$REPO_ROOT/labs/lab-3.1/manifests/config-secret-pod.yaml"

kubectl wait \
    --for=condition=Ready \
    pod/movie-service-flags-demo \
    -n "$NAMESPACE" \
    --timeout=60s

echo
echo "[4/6] Verify Volume Mount"

kubectl exec movie-service-flags-demo -n "$NAMESPACE" -- \
    ls -la /etc/movie-service/feature-flags

echo
echo "--- MAX_HOLD_SEATS_PER_USER ---"

kubectl exec movie-service-flags-demo -n "$NAMESPACE" -- \
    cat /etc/movie-service/feature-flags/MAX_HOLD_SEATS_PER_USER

echo
echo "--- Secret Env ---"

kubectl exec movie-service-flags-demo -n "$NAMESPACE" -- \
    env | grep DB_PASSWORD

echo
echo "[5/6] Update ConfigMap..."

kubectl patch configmap movie-service-feature-flags -n "$NAMESPACE" \
    -p '{"data":{"MAX_HOLD_SEATS_PER_USER":"10"}}'

echo
echo "Waiting 60 seconds for kubelet sync..."

sleep 60

echo
echo "--- File After Update ---"

kubectl exec movie-service-flags-demo -n "$NAMESPACE" -- \
    cat /etc/movie-service/feature-flags/MAX_HOLD_SEATS_PER_USER

echo
echo "Updating Secret..."

kubectl create secret generic movie-service-db-secret -n "$NAMESPACE" \
    --from-literal=DB_PASSWORD="${LAB_DB_PASSWORD}_rotated" \
    --dry-run=client -o yaml | kubectl apply -f -

echo
echo "--- Env After Secret Update (should stay old value) ---"

kubectl exec movie-service-flags-demo -n "$NAMESPACE" -- \
    env | grep DB_PASSWORD

echo
echo "========== LAB COMPLETED =========="
