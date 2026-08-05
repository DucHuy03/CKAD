#!/usr/bin/env bash
# Applies isolation policies; use a policy-capable CNI and label test Pods first.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/run-lab-manifests.sh" 4.3
