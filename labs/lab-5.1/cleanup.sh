#!/usr/bin/env bash
# Removes the self-healing Deployment only.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/cleanup-lab-manifests.sh" 5.1
