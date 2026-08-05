#!/bin/bash

set -e

# Các image demo đã được build với tag 1.0.0, không dùng :latest.
IMAGE_TAG="${IMAGE_TAG:-1.0.0}"

echo "========================================"
echo " Lab 1.4 - Label & Annotation Drill"
echo "========================================"

echo "[1/8] Switching namespace to lab..."
kubectl config set-context --current --namespace=lab

echo
echo "========== Creating Debug Pods =========="

for svc in movie-service booking-service payment-service notification-service api-gateway
do
    kubectl run debug-$svc \
        --image="$svc:$IMAGE_TAG" \
        --restart=Never \
        --labels="release=v1-2-0,component=$svc,purpose=readiness-check"
done

echo
kubectl get pods --show-labels

############################################################

echo
echo "========== Label Selectors =========="

echo
echo "--- release=v1-2-0 ---"
kubectl get pods -l release=v1-2-0

echo
echo "--- booking-service ---"
kubectl get pods -l component=booking-service

echo
echo "--- movie, booking, payment ---"
kubectl get pods -l 'component in (movie-service,booking-service,payment-service)'

echo
echo "--- release + purpose ---"
kubectl get pods -l 'release=v1-2-0,purpose=readiness-check'

############################################################

echo
echo "========== Step 3 (Expected Error) =========="

set +e
kubectl label pods -l release=v1-2-0 release=v1-3-0
set -e

echo
echo "(Nếu xuất hiện lỗi overwrite is false... thì đúng theo yêu cầu của lab)"

############################################################

echo
echo "========== Overwrite Label =========="

kubectl label pods -l release=v1-2-0 release=v1-3-0 --overwrite

echo
echo "--- Pods with release=v1-3-0 ---"
kubectl get pods -l release=v1-3-0

echo
echo "--- Pods with release=v1-2-0 (should be empty) ---"
kubectl get pods -l release=v1-2-0

############################################################

echo
echo "========== Add Status Label =========="

kubectl label pods \
    -l purpose=readiness-check \
    status=verified \
    --overwrite

############################################################

echo
echo "========== Annotation =========="

kubectl annotate pods \
    -l component=payment-service \
    checked-by="ban" \
    note="Da smoke test thu cong truoc release v1.3.0" \
    --overwrite

echo
kubectl describe pod debug-payment-service | grep -A5 Annotations

############################################################

echo
echo "========== Cleanup =========="

kubectl delete pods -l purpose=readiness-check

echo
echo "--- Remaining Pods ---"
kubectl get pods -l release=v1-3-0

echo
echo "========== LAB COMPLETED =========="
