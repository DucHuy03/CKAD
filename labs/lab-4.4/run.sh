#!/usr/bin/env bash
# Applies PVC and writer Pod; delete/recreate only the Pod to test persistence.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/run-lab-manifests.sh" 4.4
