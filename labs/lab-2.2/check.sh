#!/usr/bin/env bash
# Stable, read-only checker for Lab 2.2; it verifies blue/green resources.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/check-lab.sh" 2.2
