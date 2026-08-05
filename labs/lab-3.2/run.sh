#!/usr/bin/env bash

# Stop on errors, unset variables, and failed commands in pipes.
set -euo pipefail

# Resolve the locked-down Pod manifest from this script, not the caller CWD.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

NAMESPACE="lab"
POD="movie-service-locked-down"
DEMO_STARTED=false

restore_original_state() {
    local exit_code=$?

    # Before the demo starts, nothing was created, so no cleanup is needed.
    if [[ "$DEMO_STARTED" != true ]]; then
        exit "$exit_code"
    fi

    echo
    echo "Cleaning up Lab 3.2 demo resources..."
    kubectl delete pod "$POD" -n "$NAMESPACE" --ignore-not-found >/dev/null || true
    exit "$exit_code"
}
trap restore_original_state EXIT INT TERM

echo "========================================"
echo " Lab 3.2 - Security Context Lockdown"
echo "========================================"

# From here, the script creates a Pod in the cluster; the trap must delete it.
DEMO_STARTED=true

echo
echo "[1/5] Applying manifest..."

kubectl apply -n "$NAMESPACE" -f "$REPO_ROOT/labs/lab-3.2/manifests/locked-down-pod.yaml"

echo
echo "Waiting for Pod..."

kubectl wait \
    --for=condition=Ready \
    pod/"$POD" \
    -n "$NAMESPACE" \
    --timeout=60s

echo
kubectl get pod "$POD" -n "$NAMESPACE" -o wide

echo
echo "[2/5] Verify ReadOnly Root Filesystem"

echo
echo "--- touch /test-file (expected: Read-only file system) ---"

set +e
kubectl exec "$POD" -n "$NAMESPACE" -- touch /test-file
set -e

echo
echo "--- App Log ---"

kubectl exec "$POD" -n "$NAMESPACE" -- \
    cat /var/log/app/app.log

echo
echo "--- Current User ---"

kubectl exec "$POD" -n "$NAMESPACE" -- id

echo
echo "[3/5] Verify Linux Capabilities"

kubectl exec "$POD" -n "$NAMESPACE" -- \
    cat /proc/1/status | grep Cap

echo
echo "========== LAB COMPLETED =========="
