#!/usr/bin/env bash
# Demo Lab 1.2 can DB password trung voi PostgreSQL dang chay. Runner tao moi
# Pod moi lan chay de init container va sidecar luon duoc quan sat tu dau.

set -euo pipefail

# Resolve manifests from this script, so the command works from any directory.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
NAMESPACE="lab"
POD="movie-catalog-multi"
: "${LAB_DB_PASSWORD:?Set LAB_DB_PASSWORD before running Lab 1.2}"

echo "========================================"
echo " Lab 1.2 - Init + Sidecar Pattern"
echo "========================================"

echo "[1/5] Switching namespace to lab..."
kubectl config set-context --current --namespace="$NAMESPACE"

# Secret o namespace lab khong luu gia tri trong Git.
NAMESPACE="$NAMESPACE" bash "$REPO_ROOT/scripts/create-lab-secrets.sh"

echo
echo "[2/5] Recreating Pod..."
# Pod Failed/Init:CrashLoopBackOff khong duoc kubectl apply tao lai.
kubectl delete pod "$POD" --ignore-not-found --wait=true
kubectl apply -f "$REPO_ROOT/labs/lab-1.2/manifests/init-sidecar-pod.yaml"

echo
echo "[3/5] Waiting for Pod to become Ready..."
kubectl wait --for=condition=Ready "pod/$POD" --timeout=120s

echo
echo "========== POD STATUS =========="
kubectl get pod "$POD" -o wide

echo
echo "========== INIT CONTAINER LOG =========="
kubectl logs "$POD" -c init-db-migrate

echo
echo "========== APP CONTAINER LOG =========="
kubectl logs "$POD" -c app

echo
echo "========== SIDECAR LOG =========="
kubectl logs "$POD" -c log-shipper

echo
echo "========== DONE =========="
