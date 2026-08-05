#!/usr/bin/env bash

# Stop on errors, unset variables, and failed commands in pipes.
set -euo pipefail

# Kustomize files are resolved from the repository root, not the caller CWD.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
KUSTOMIZE_DIR="$REPO_ROOT/movie-booking/k8s-kustomize"
NAMESPACE="lab"

# Snapshot the dev overlay so the EXIT trap can put it back byte-for-byte,
# regardless of which step the demo stops at (step 4 edits its image tag).
DEV_KUSTOMIZATION="$KUSTOMIZE_DIR/overlays/dev/kustomization.yaml"
ORIGINAL_DEV_KUSTOMIZATION="$(cat "$DEV_KUSTOMIZATION")"
DEMO_STARTED=false

restore_original_state() {
    local exit_code=$?

    # Before the demo starts, nothing was applied to the cluster or edited on
    # disk, so no restoration is needed.
    if [[ "$DEMO_STARTED" != true ]]; then
        exit "$exit_code"
    fi

    echo
    echo "Restoring Lab 2.4 overlay and cluster state..."
    printf '%s' "$ORIGINAL_DEV_KUSTOMIZATION" > "$DEV_KUSTOMIZATION"

    kubectl delete -k "$KUSTOMIZE_DIR/overlays/dev" -n "$NAMESPACE" --ignore-not-found >/dev/null || true
    kubectl delete -k "$KUSTOMIZE_DIR/overlays/prod" -n "$NAMESPACE" --ignore-not-found >/dev/null || true
    exit "$exit_code"
}
trap restore_original_state EXIT INT TERM

echo "========================================"
echo " Lab 2.4 - Kustomize Overlay"
echo "========================================"

echo
echo "[0/5] Checking local movie-service:v1 / movie-service:v2 images..."
for tag in v1 v2; do
    if ! docker image inspect "movie-service:$tag" >/dev/null 2>&1; then
        echo "movie-service:$tag not found locally. Run labs/lab-2.1/run.sh first to build it."
        exit 1
    fi
done

# From here, the script edits overlays/dev and applies to the cluster; the
# trap must restore both.
DEMO_STARTED=true

cd "$KUSTOMIZE_DIR"

echo
echo "[1/5] Build DEV overlay"

kubectl kustomize overlays/dev

echo
echo "========================================"

echo
echo "[1/5] Build PROD overlay"

kubectl kustomize overlays/prod

echo
echo "[2/5] Diff DEV vs PROD"

diff \
    <(kubectl kustomize overlays/dev) \
    <(kubectl kustomize overlays/prod) || true

echo
echo "[3/5] Apply overlays"

kubectl apply -k overlays/dev
kubectl apply -k overlays/prod

kubectl rollout status deployment/dev-movie-service -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/prod-movie-service -n "$NAMESPACE" --timeout=120s

echo
echo "--- Deployments ---"

kubectl get deployments \
    -n "$NAMESPACE" \
    -l app=movie-service

echo
echo "[4/5] Change DEV image v1 -> v2"

sed -i 's/newTag: v1/newTag: v2/' overlays/dev/kustomization.yaml

kubectl apply -k overlays/dev

kubectl rollout status \
    deployment/dev-movie-service \
    -n "$NAMESPACE" \
    --timeout=120s

echo
echo "Restore DEV image v2 -> v1"

sed -i 's/newTag: v2/newTag: v1/' overlays/dev/kustomization.yaml

kubectl apply -k overlays/dev

kubectl rollout status \
    deployment/dev-movie-service \
    -n "$NAMESPACE" \
    --timeout=120s

echo
echo "========== LAB COMPLETED =========="
