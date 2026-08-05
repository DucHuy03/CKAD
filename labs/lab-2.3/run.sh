#!/usr/bin/env bash

# Stop on errors, unset variables, and failed commands in pipes.
set -euo pipefail

# Resolve paths once so the script works from any current directory.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
NAMESPACE="movie-booking"
DEPLOYMENT="booking-service"
HPA_MANIFEST="$REPO_ROOT/movie-booking/k8s/autoscaling/booking-service-hpa.yaml"

# The main system already ships a permanent HPA for booking-service (see
# HPA_MANIFEST, applied by scripts/deploy-kubernetes.sh). Lab 2.3 replaces it
# temporarily with its own min=1/max=10 demo HPA, so the EXIT trap below must
# always put the original HPA and replica count back.
ORIGINAL_REPLICAS="$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')"
DEMO_STARTED=false

show_rollout_diagnostics() {
    echo
    echo "--- booking-service Pods ---"
    kubectl get pods -n "$NAMESPACE" -l app=booking-service -o wide || true
    echo "--- Recent events ---"
    kubectl get events -n "$NAMESPACE" --sort-by=.lastTimestamp | tail -n 30 || true
}

restore_original_state() {
    local exit_code=$?

    # Before the demo starts, no live workload was changed, so no restoration
    # is needed. This avoids creating a needless rollout on preflight fail.
    if [[ "$DEMO_STARTED" != true ]]; then
        exit "$exit_code"
    fi

    echo
    echo "Restoring booking-service to its state before Lab 2.3..."
    kubectl delete pod load-generator -n "$NAMESPACE" --ignore-not-found >/dev/null || true

    # Drop the lab's demo HPA and put the checked-in production HPA back,
    # even when the user stops the demo with Ctrl+C.
    kubectl delete hpa booking-service -n "$NAMESPACE" --ignore-not-found >/dev/null || true
    kubectl apply -f "$HPA_MANIFEST" >/dev/null || true

    kubectl scale "deployment/$DEPLOYMENT" --replicas="$ORIGINAL_REPLICAS" -n "$NAMESPACE" >/dev/null || true
    kubectl rollout status "deployment/$DEPLOYMENT" -n "$NAMESPACE" --timeout=120s || true
    exit "$exit_code"
}
trap restore_original_state EXIT INT TERM

echo "========================================"
echo " Lab 2.3 - Horizontal Pod Autoscaler"
echo "========================================"

echo
echo "[0/5] Checking the original booking-service deployment..."
if ! kubectl rollout status "deployment/$DEPLOYMENT" -n "$NAMESPACE" --timeout=60s; then
    echo "booking-service is not healthy before Lab 2.3; the demo was not started."
    echo "Fix the baseline deployment first, then run this lab again."
    show_rollout_diagnostics
    exit 1
fi

# From here, the script may modify the Deployment/HPA and the trap must restore them.
DEMO_STARTED=true

echo
echo "[1/5] Installing metrics-server..."

kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

kubectl patch deployment metrics-server \
    -n kube-system \
    --type='json' \
    -p='[
        {
            "op":"add",
            "path":"/spec/template/spec/containers/0/args/-",
            "value":"--kubelet-insecure-tls"
        }
    ]' || true

echo
echo "Waiting for metrics-server..."

kubectl rollout status deployment/metrics-server \
    -n kube-system \
    --timeout=120s

echo
echo "--- Node Metrics ---"
kubectl top nodes

echo
echo "[2/5] Scaling to 10 replicas..."

kubectl scale "deployment/$DEPLOYMENT" \
    --replicas=10 \
    -n "$NAMESPACE"

kubectl rollout status "deployment/$DEPLOYMENT" \
    -n "$NAMESPACE"

echo
echo "--- Pods ---"
kubectl get pods \
    -n "$NAMESPACE" \
    -l app=booking-service

echo
echo "--- Endpoints ---"
kubectl get endpoints booking-service \
    -n "$NAMESPACE"

echo
echo "Scaling back to 1..."

kubectl scale "deployment/$DEPLOYMENT" \
    --replicas=1 \
    -n "$NAMESPACE"

kubectl rollout status "deployment/$DEPLOYMENT" \
    -n "$NAMESPACE"

echo
echo "[3/5] Creating HPA..."

# The production HPA (same name, applied by scripts/deploy-kubernetes.sh)
# would otherwise collide with kubectl autoscale below; swap it out for the
# demo HPA and let the trap put the original back on exit.
kubectl delete hpa booking-service -n "$NAMESPACE" --ignore-not-found

kubectl autoscale deployment booking-service \
    -n "$NAMESPACE" \
    --cpu=50% \
    --min=1 \
    --max=10

echo
kubectl get hpa booking-service -n "$NAMESPACE"

echo
echo "[4/5] Creating load generator..."

kubectl delete pod load-generator -n "$NAMESPACE" --ignore-not-found

kubectl run load-generator \
    --image=busybox \
    --restart=Never \
    -n "$NAMESPACE" \
    -- \
    /bin/sh -c \
    "while true; do wget -q -O- http://booking-service:8080/healthz; done"

sleep 10

echo
echo "--- HPA ---"
kubectl get hpa booking-service -n "$NAMESPACE"

echo
echo "--- Pod CPU ---"
kubectl top pods \
    -n "$NAMESPACE" \
    -l app=booking-service || true

echo
echo "========== FINAL STATUS =========="

kubectl get deployment booking-service -n "$NAMESPACE"
kubectl get hpa booking-service -n "$NAMESPACE"
kubectl get pods -n "$NAMESPACE" -l app=booking-service

echo
echo "========== LAB COMPLETED =========="
