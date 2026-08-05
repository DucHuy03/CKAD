#!/usr/bin/env bash
# Deletes only resources declared by one lab. Usage: bash scripts/cleanup-lab-manifests.sh 4.1
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LAB_ID="${1:?Provide a lab id, for example 4.1}"
kubectl delete -f "$ROOT/labs/lab-$LAB_ID/manifests" --ignore-not-found
