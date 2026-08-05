#!/usr/bin/env bash
# Stable, read-only checker for Lab 1.1; implementation stays at repository root in Phase 1.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/check-lab.sh" 1.1
