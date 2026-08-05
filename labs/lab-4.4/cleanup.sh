#!/usr/bin/env bash
# Deletes Lab 4.4 resources, including the PVC; back up data first if needed.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/cleanup-lab-manifests.sh" 4.4
