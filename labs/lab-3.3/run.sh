#!/usr/bin/env bash

# Stop on errors, unset variables, and failed commands in pipes.
set -euo pipefail

# Work beside this script so each manifest name is resolved reliably.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ASSET_DIR="$REPO_ROOT/movie-booking/lab-3.3"

NAMESPACE="lab"
SA="movie-reader"
POD="movie-api-client"
DEMO_STARTED=false

restore_original_state() {
    local exit_code=$?
    if [[ "$DEMO_STARTED" != true ]]; then
        exit "$exit_code"
    fi

    echo
    echo "Cleaning up Lab 3.3 demo resources..."
    kubectl delete pod "$POD" -n "$NAMESPACE" --ignore-not-found >/dev/null || true
    exit "$exit_code"
}
trap restore_original_state EXIT INT TERM

echo "========================================"
echo " Lab 3.3 - ServiceAccount & RBAC"
echo "========================================"

echo
echo "[1/9] Applying ServiceAccount, Role, RoleBinding..."
kubectl apply -f "$ASSET_DIR/serviceaccount.yaml"
kubectl apply -f "$ASSET_DIR/role.yaml"
kubectl apply -f "$ASSET_DIR/rolebinding.yaml"

DEMO_STARTED=true

echo
echo "[2/9] Applying Pod..."
kubectl apply -n "$NAMESPACE" -f "$REPO_ROOT/labs/lab-3.3/manifests/api-client-pod.yaml"

kubectl wait \
    --for=condition=Ready \
    pod/"$POD" \
    -n "$NAMESPACE" \
    --timeout=60s

echo
echo "[3/9] Verify ServiceAccount"

kubectl describe pod "$POD" -n "$NAMESPACE" | grep "Service Account"

echo
echo "[4/9] Roles"

kubectl get role -n "$NAMESPACE"

echo
echo "[5/9] RoleBindings"

kubectl get rolebinding -n "$NAMESPACE"

echo
echo "[6/9] RBAC Verification"

echo
echo "--- Can list pods (expected: yes) ---"

kubectl auth can-i list pods \
    --as="system:serviceaccount:$NAMESPACE:$SA" \
    -n "$NAMESPACE"

# "kubectl auth can-i" exits 1 when the answer is "no" - that IS the expected
# outcome for the next two checks, so temporarily disable set -e around them
# rather than letting the whole script die on a correctly-denied check.
echo
echo "--- Can delete pods (expected: no) ---"

set +e
kubectl auth can-i delete pods \
    --as="system:serviceaccount:$NAMESPACE:$SA" \
    -n "$NAMESPACE"
set -e

echo
echo "--- Can read secrets (expected: no) ---"

set +e
kubectl auth can-i get secrets \
    --as="system:serviceaccount:$NAMESPACE:$SA" \
    -n "$NAMESPACE"
set -e

echo
echo "[7/9] Verify ServiceAccount Token"

kubectl get pod "$POD" -n "$NAMESPACE" -o yaml | grep -A10 projected

echo
echo "========== LAB COMPLETED =========="
