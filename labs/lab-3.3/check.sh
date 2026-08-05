#!/usr/bin/env bash
# Stable, read-only checker for Lab 3.3; it verifies RBAC runner and client Pod evidence.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/check-lab.sh" 3.3
