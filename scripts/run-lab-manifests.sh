#!/usr/bin/env bash
# Applies all manifests of one lab. Usage: bash scripts/run-lab-manifests.sh 4.1
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LAB_ID="${1:?Provide a lab id, for example 4.1}"
# Each lab owns a manifest directory; this keeps cleanup and review scoped.
kubectl apply -f "$ROOT/labs/lab-$LAB_ID/manifests"
