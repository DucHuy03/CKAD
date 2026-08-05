#!/usr/bin/env bash
# Apply the broken Service, inspect zero endpoints, then apply the fixed pair.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
kubectl apply -f "$ROOT/labs/lab-5.3/manifests/broken-service.yaml"
kubectl get endpoints -n lab lab53-broken
kubectl apply -f "$ROOT/labs/lab-5.3/manifests/fixed-deployment.yaml"
kubectl rollout status -n lab deployment/lab53-fixed
kubectl get endpoints -n lab lab53-fixed
