#!/usr/bin/env bash
# Read-only observability drill; run Lab 5.1 first to ensure a target exists.
set -euo pipefail
NAMESPACE="${NAMESPACE:-lab}"
POD="$(kubectl get pod -n "$NAMESPACE" -l app=lab51-self-healing -o jsonpath='{.items[0].metadata.name}')"
kubectl logs -n "$NAMESPACE" "$POD" -c app --tail=20
kubectl describe pod -n "$NAMESPACE" "$POD"
kubectl get events -n "$NAMESPACE" --sort-by=.lastTimestamp
kubectl top pod -n "$NAMESPACE" "$POD"
