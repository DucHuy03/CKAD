#!/usr/bin/env bash
# Deploys the probe exercise. Delete /tmp/ready to observe readiness change.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/run-lab-manifests.sh" 5.1
