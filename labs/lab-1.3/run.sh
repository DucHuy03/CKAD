#!/bin/bash

set -e

# The solution manifests live under movie-booking; never depend on caller CWD.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "========================================"
echo " Lab 1.3 - Job & CronJob"
echo "========================================"

echo "[1/7] Switching namespace to lab..."
kubectl config set-context --current --namespace=lab

####################################################
# PART 1 - JOB
####################################################

echo
echo "========== APPLY JOB =========="

kubectl apply -f "$REPO_ROOT/labs/lab-1.3/manifests/database-migration-job.yaml"

echo
echo "Waiting for Job to complete..."
kubectl wait --for=condition=complete job/movie-db-migrate-job --timeout=120s

echo
echo "--- Job Status ---"
kubectl get jobs

echo
echo "--- Job Pod ---"
kubectl get pods -l job-name=movie-db-migrate-job

echo
echo "--- Job Logs ---"
kubectl logs -l job-name=movie-db-migrate-job

####################################################
# PART 2 - CRONJOB
####################################################

echo
echo "========== APPLY CRONJOB =========="

kubectl apply -f "$REPO_ROOT/labs/lab-1.3/manifests/expired-hold-cleanup-cronjob.yaml"

echo
echo "--- CronJob ---"
kubectl get cronjob booking-cleanup-cronjob

echo
echo "Creating manual Job from CronJob..."

kubectl create job \
  --from=cronjob/booking-cleanup-cronjob \
  booking-cleanup-manual-run

echo
echo "Waiting for manual Job..."
kubectl wait \
  --for=condition=complete \
  job/booking-cleanup-manual-run \
  --timeout=120s

echo
echo "--- Manual Job Pod ---"
kubectl get pods -l job-name=booking-cleanup-manual-run

echo
echo "--- Manual Job Logs ---"
kubectl logs -l job-name=booking-cleanup-manual-run

####################################################
# COMPARISON
####################################################

echo
echo "========== JOB POD =========="
kubectl get pods -l job-name=movie-db-migrate-job

echo
echo "========== DEPLOYMENT POD =========="
kubectl get pods -n movie-booking -l app=movie-service

echo
echo "========== LAB COMPLETED =========="
