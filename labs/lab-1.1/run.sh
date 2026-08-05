#!/usr/bin/env bash
# Demo Lab 1.1 tao MOI debug Pod moi lan chay. LAB_DB_PASSWORD phai trung voi
# DB_PASSWORD cua movie-service dang chay trong namespace movie-booking.

set -euo pipefail

# Resolve manifests from this script, so the command works from any directory.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
NAMESPACE="lab"
POD="movie-catalog-debug"
: "${LAB_DB_PASSWORD:?Set LAB_DB_PASSWORD before running Lab 1.1}"

echo "========================================"
echo " Lab - Movie Catalog Debug Pod"
echo "========================================"

echo "[1/5] Switching namespace to lab..."
kubectl config set-context --current --namespace="$NAMESPACE"

# Tao Secret rieng trong namespace lab; password khong bao gio duoc commit.
NAMESPACE="$NAMESPACE" bash "$REPO_ROOT/scripts/create-lab-secrets.sh"

echo
echo "[2/5] Recreating debug Pod..."
# kubectl apply khong tao lai Pod da Failed, nen can xoa fixture cu truoc.
kubectl delete pod "$POD" --ignore-not-found --wait=true
kubectl apply -f "$REPO_ROOT/labs/lab-1.1/manifests/debug-pod.yaml"

echo
echo "[3/5] Waiting for Pod to be created..."
kubectl wait --for=condition=Ready "pod/$POD" --timeout=90s

echo
echo "========== VERIFY =========="

echo
echo "--- Pod Information ---"
kubectl get pod "$POD" -o wide

echo
echo "--- Labels ---"
kubectl get pod "$POD" --show-labels

echo
echo "--- Environment Variables ---"
kubectl get pod "$POD" \
-o jsonpath='{.spec.containers[0].env}'
echo

echo
echo "--- Resources ---"
kubectl get pod "$POD" \
-o jsonpath='{.spec.containers[0].resources}'
echo

echo
echo "--- Status ---"
kubectl get pod "$POD" \
-o jsonpath='{.status.phase}'
echo

echo
echo "[4/5] Port Forward..."
kubectl port-forward "pod/$POD" 8080:8080 >/dev/null 2>&1 &
PF_PID=$!
trap 'kill "$PF_PID" 2>/dev/null || true' EXIT

# Chi tiep tuc khi port-forward van song; tranh curl nham mot process cu tren
# localhost:8080 trong truong hop port-forward that bai.
sleep 2
kill -0 "$PF_PID"

echo
echo "--- Calling /ready ---"
curl --fail --silent --show-error http://localhost:8080/ready

echo
echo "[5/5] Stopping Port Forward..."
kill "$PF_PID" 2>/dev/null || true
trap - EXIT

echo
echo "========== DONE =========="
