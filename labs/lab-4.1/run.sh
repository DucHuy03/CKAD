#!/usr/bin/env bash
# Runs Lab 4.1 resources, then inspect endpoints with kubectl get endpoints -n lab.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/run-lab-manifests.sh" 4.1
