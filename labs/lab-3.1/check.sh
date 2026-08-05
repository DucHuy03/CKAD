#!/usr/bin/env bash
# Stable, read-only checker for Lab 3.1; it checks ConfigMap/Secret exercise assets.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/check-lab.sh" 3.1
