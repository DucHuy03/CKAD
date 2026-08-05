#!/usr/bin/env bash
# Deletes the intentionally broken Service and its fixed counterpart.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/cleanup-lab-manifests.sh" 5.3
