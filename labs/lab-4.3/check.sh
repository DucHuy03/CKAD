#!/usr/bin/env bash
# Stable checker for Lab 4.3; NetworkPolicy assertions are planned for Phase 2.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/check-lab.sh" 4.3
