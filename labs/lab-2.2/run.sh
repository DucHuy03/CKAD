#!/bin/bash

set -e

# Blue/green manifests are stored in movie-booking; make all paths deterministic.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
MANIFEST_DIR="$REPO_ROOT/labs/lab-2.2/manifests"

NAMESPACE="lab"
SERVICE="movie-service-bluegreen"

echo "========================================"
echo " Lab 2.2 - Blue/Green Deployment"
echo "========================================"

####################################################
# STEP 1 - APPLY
####################################################

echo
echo "[1/5] Applying manifests..."

kubectl apply -f "$MANIFEST_DIR/blue-deployment.yaml"
kubectl apply -f "$MANIFEST_DIR/green-deployment.yaml"
kubectl apply -f "$MANIFEST_DIR/active-service.yaml"

echo
echo "Waiting for Pods..."

kubectl wait --for=condition=Ready pod \
    -l app=movie-service-bluegreen \
    -n $NAMESPACE \
    --timeout=120s

echo
echo "========== PODS =========="
kubectl get pods \
    -n $NAMESPACE \
    -l app=movie-service-bluegreen \
    --show-labels

####################################################
# STEP 2 - BLUE
####################################################

echo
echo "[2/5] Service currently points to BLUE..."

kubectl port-forward svc/$SERVICE 8080:8080 -n $NAMESPACE >/dev/null 2>&1 &
PF_PID=$!

sleep 3

echo
echo "--- Health Check (BLUE) ---"
curl http://localhost:8080/healthz

####################################################
# STEP 3 - GREEN
####################################################

echo
echo "[3/5] Switching to GREEN..."

kubectl patch service $SERVICE \
    -n $NAMESPACE \
    -p '{"spec":{"selector":{"version":"green"}}}'

sleep 2

echo
echo "--- Health Check (GREEN) ---"
curl http://localhost:8080/healthz

####################################################
# STEP 4 - ROLLBACK
####################################################

echo
echo "[4/5] Rolling back to BLUE..."

kubectl patch service $SERVICE \
    -n $NAMESPACE \
    -p '{"spec":{"selector":{"version":"blue"}}}'

sleep 2

echo
echo "--- Health Check (BLUE) ---"
curl http://localhost:8080/healthz

####################################################
# STEP 5 - CLEANUP
####################################################

echo
echo "[5/5] Cleanup..."

kill $PF_PID 2>/dev/null || true

kubectl delete -f "$MANIFEST_DIR/blue-deployment.yaml" --ignore-not-found
kubectl delete -f "$MANIFEST_DIR/green-deployment.yaml" --ignore-not-found
kubectl delete -f "$MANIFEST_DIR/active-service.yaml" --ignore-not-found

echo
echo "========== LAB COMPLETED =========="
