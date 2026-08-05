#!/bin/bash
#
# Lab 3.4 — Namespace Quotas
#
# Muc tieu:
#   1. Tao ResourceQuota de gioi han tong tai nguyen namespace.
#   2. Tao LimitRange de dat request/limit mac dinh cho moi container.
#   3. Tao Pod KHONG khai bao resources -> Kubernetes tu them.
#   4. Tao Pod vuot qua gioi han -> Kubernetes tu choi.
#   5. Tao nhieu Pod de quan sat quota bi tieu thu.
#

set -e

# Work beside this script so each manifest name is resolved reliably.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ASSET_DIR="$REPO_ROOT/movie-booking/lab-3.4"

echo "======================================================="
echo "Lab 3.4 - Namespace Quotas"
echo "======================================================="

#
# Su dung namespace lab
#
kubectl config set-context --current --namespace=lab

echo
echo "=== Buoc 1. Tao ResourceQuota ==="

kubectl apply -f "$ASSET_DIR/resourcequota.yaml"

echo
kubectl describe resourcequota lab-quota

echo
echo "=== Buoc 2. Tao LimitRange ==="

kubectl apply -f "$ASSET_DIR/limitrange.yaml"

echo
kubectl describe limitrange lab-defaults

echo
echo "=== Buoc 3. Tao Pod KHONG khai bao resources ==="

kubectl apply -f "$ASSET_DIR/pod-default.yaml" || true

echo
kubectl get pod quota-demo-default

echo
echo "Neu Pod tao thanh cong, kiem tra resources da duoc them mac dinh"

kubectl describe pod quota-demo-default || true

echo
echo "=== Buoc 4. Tao Pod vuot max cua LimitRange ==="

kubectl apply -f "$REPO_ROOT/labs/lab-3.4/manifests/quota-exceeded-pod.yaml" || true

echo
echo "Neu thay Forbidden -> Day la ket qua dung."

echo
echo "=== Buoc 5. Tao nhieu Pod de tieu thu quota ==="

for i in $(seq 1 12)
do
    kubectl run quota-demo-$i \
      --image=nginx:1.27-alpine \
      --restart=Never \
      --image-pull-policy=IfNotPresent || true
done

echo
echo "=== Buoc 6. Kiem tra quota ==="

kubectl describe resourcequota lab-quota

echo
echo "=== Danh sach Pod ==="

kubectl get pods

echo
echo "Lab 3.4 hoan thanh."
