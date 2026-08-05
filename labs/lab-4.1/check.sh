#!/usr/bin/env bash
# Stable checker for Lab 4.1; it will become a cluster assertion in Phase 2.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/check-lab.sh" 4.1
