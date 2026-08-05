#!/usr/bin/env bash

# Stop on errors, unset variables, and failed commands in pipes.
set -euo pipefail

# Resolve paths once so the script works from any current directory.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
NAMESPACE="movie-booking"
DEPLOYMENT="movie-service"

# Remember the real application state before the demo. Lab 2.1 intentionally
# deploys a non-existent image; the EXIT trap below always restores this state.
ORIGINAL_IMAGE="$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[?(@.name=="app")].image}')"
ORIGINAL_APP_VERSION="$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" -o jsonpath='{range .spec.template.spec.containers[?(@.name=="app")].env[?(@.name=="APP_VERSION")]}{.value}{end}')"
DEMO_STARTED=false

show_rollout_diagnostics() {
    # Keep the most useful Kubernetes evidence together when a Pod is not Ready.
    echo
    echo "--- movie-service Pods ---"
    kubectl get pods -n "$NAMESPACE" -l app=movie-service -o wide || true
    echo "--- Recent events ---"
    kubectl get events -n "$NAMESPACE" --sort-by=.lastTimestamp | tail -n 30 || true
    echo "--- Application and sidecar logs ---"
    kubectl logs "deployment/$DEPLOYMENT" -n "$NAMESPACE" --all-containers --tail=80 || true
}

restore_original_deployment() {
    local exit_code=$?

    # Before the demo starts, no live workload was changed, so no restoration
    # is needed. This avoids creating a needless new rollout on preflight fail.
    if [[ "$DEMO_STARTED" != true ]]; then
        exit "$exit_code"
    fi

    # Do not leave movie-service on v1/v2 or the deliberately broken image,
    # even when the user stops the demo with Ctrl+C.
    echo
    echo "Restoring movie-service to its state before Lab 2.1..."
    kubectl set image "deployment/$DEPLOYMENT" "app=$ORIGINAL_IMAGE" -n "$NAMESPACE" >/dev/null || true

    if [[ -n "$ORIGINAL_APP_VERSION" ]]; then
        kubectl set env "deployment/$DEPLOYMENT" "APP_VERSION=$ORIGINAL_APP_VERSION" -n "$NAMESPACE" >/dev/null || true
    else
        kubectl set env "deployment/$DEPLOYMENT" "APP_VERSION-" -n "$NAMESPACE" >/dev/null || true
    fi

    kubectl rollout status "deployment/$DEPLOYMENT" -n "$NAMESPACE" --timeout=180s || true
    exit "$exit_code"
}
trap restore_original_deployment EXIT INT TERM

echo "========================================"
echo " Lab 2.1 - Rolling Update & Rollback"
echo "========================================"

echo
echo "[0/6] Checking the original movie-service deployment..."
if ! kubectl rollout status "deployment/$DEPLOYMENT" -n "$NAMESPACE" --timeout=60s; then
    echo "movie-service is not healthy before Lab 2.1; the demo was not started."
    echo "Fix the baseline deployment first, then run this lab again."
    show_rollout_diagnostics
    exit 1
fi

# From here, the script may modify the Deployment and the trap must restore it.
DEMO_STARTED=true

echo
echo "[1/6] Building two local demo image tags..."
docker build -t movie-service:v1 "$REPO_ROOT/movie-booking/movie-service"
docker build -t movie-service:v2 "$REPO_ROOT/movie-booking/movie-service"

echo
echo "[2/6] Deploying version v1..."
kubectl set image "deployment/$DEPLOYMENT" app=movie-service:v1 -n "$NAMESPACE"
kubectl set env "deployment/$DEPLOYMENT" APP_VERSION=v1 -n "$NAMESPACE"
if ! kubectl rollout status "deployment/$DEPLOYMENT" -n "$NAMESPACE" --timeout=180s; then
    echo "Version v1 did not become Ready."
    show_rollout_diagnostics
    exit 1
fi

# This checks the NodePort gateway is reachable. The rollout command above
# verifies that the movie-service Pods themselves became Ready.
echo "--- Gateway health check (v1) ---"
curl --fail --silent --show-error http://localhost:30080/healthz
echo

echo
echo "[3/6] Rolling update to v2..."
kubectl set image "deployment/$DEPLOYMENT" app=movie-service:v2 -n "$NAMESPACE"
kubectl set env "deployment/$DEPLOYMENT" APP_VERSION=v2 -n "$NAMESPACE"
if ! kubectl rollout status "deployment/$DEPLOYMENT" -n "$NAMESPACE" --timeout=180s; then
    echo "Version v2 did not become Ready."
    show_rollout_diagnostics
    exit 1
fi

echo "--- ReplicaSets ---"
kubectl get rs -n "$NAMESPACE" -l app=movie-service
echo "--- Gateway health check (v2) ---"
curl --fail --silent --show-error http://localhost:30080/healthz
echo

echo
echo "[4/6] Simulating a broken deployment (expected to time out)..."
kubectl set image "deployment/$DEPLOYMENT" app=movie-service:v3-typo-khong-ton-tai -n "$NAMESPACE"

# A failed rollout is the intended lesson, so handle it explicitly rather than
# masking every error with `set +e`.
if kubectl rollout status "deployment/$DEPLOYMENT" -n "$NAMESPACE" --timeout=30s; then
    echo "Unexpected: the intentionally invalid image became Ready."
else
    echo "Expected: the invalid image cannot be pulled; continuing to rollback."
fi

echo "--- Current Pods ---"
kubectl get pods -n "$NAMESPACE" -l app=movie-service

echo
echo "[5/6] Rolling back the invalid revision..."
echo "--- Rollout History ---"
kubectl rollout history "deployment/$DEPLOYMENT" -n "$NAMESPACE"
kubectl rollout undo "deployment/$DEPLOYMENT" -n "$NAMESPACE"
kubectl rollout status "deployment/$DEPLOYMENT" -n "$NAMESPACE" --timeout=180s

echo "--- Gateway health check after rollback ---"
curl --fail --silent --show-error http://localhost:30080/healthz
echo

echo
echo "[6/6] Demo evidence before restoring the original deployment..."
kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE"
kubectl get rs -n "$NAMESPACE" -l app=movie-service
kubectl get pods -n "$NAMESPACE" -l app=movie-service
echo "========== LAB COMPLETED =========="
