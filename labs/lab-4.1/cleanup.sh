#!/usr/bin/env bash
# Removes only Lab 4.1 resources after the selector/endpoint exercise.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/cleanup-lab-manifests.sh" 4.1
