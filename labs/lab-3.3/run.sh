#!/bin/bash

set -e

# Work beside this script so each manifest name is resolved reliably.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ASSET_DIR="$REPO_ROOT/movie-booking/lab-3.3"

NAMESPACE="lab"
SA="movie-reader"
POD="movie-api-client"

echo "========================================"
echo " Lab 3.3 - ServiceAccount & RBAC"
echo "========================================"

####################################################
# STEP 1
####################################################

echo
echo "[1/10] Creating namespace..."

kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -

kubectl config set-context --current --namespace=$NAMESPACE

####################################################
# STEP 2-5
####################################################

echo
echo "[2/10] Applying ServiceAccount..."
kubectl apply -f "$ASSET_DIR/serviceaccount.yaml"

echo
echo "[3/10] Applying Role..."
kubectl apply -f "$ASSET_DIR/role.yaml"

echo
echo "[4/10] Applying RoleBinding..."
kubectl apply -f "$ASSET_DIR/rolebinding.yaml"

echo
echo "[5/10] Applying Pod..."
kubectl apply -f "$REPO_ROOT/labs/lab-3.3/manifests/api-client-pod.yaml"

kubectl wait \
    --for=condition=Ready \
    pod/$POD \
    --timeout=60s

####################################################
# STEP 6-8
####################################################

echo
echo "[6/10] Verify ServiceAccount"

kubectl describe pod $POD | grep "Service Account"

echo
echo "[7/10] Roles"

kubectl get role

echo
echo "[8/10] RoleBindings"

kubectl get rolebinding

####################################################
# STEP 9
####################################################

echo
echo "[9/10] RBAC Verification"

echo
echo "--- Can list pods (expected: yes) ---"

kubectl auth can-i list pods \
    --as=system:serviceaccount:$NAMESPACE:$SA \
    -n $NAMESPACE

echo
echo "--- Can delete pods (expected: no) ---"

kubectl auth can-i delete pods \
    --as=system:serviceaccount:$NAMESPACE:$SA \
    -n $NAMESPACE

echo
echo "--- Can read secrets (expected: no) ---"

kubectl auth can-i get secrets \
    --as=system:serviceaccount:$NAMESPACE:$SA \
    -n $NAMESPACE

####################################################
# STEP 10
####################################################

echo
echo "[10/10] Verify ServiceAccount Token"

kubectl get pod $POD -o yaml | grep -A10 projected

echo
echo "========== LAB COMPLETED =========="
