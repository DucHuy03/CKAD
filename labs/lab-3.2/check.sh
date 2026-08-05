#!/usr/bin/env bash
# Stable, read-only checker for Lab 3.2; it checks the locked-down Pod manifest.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/check-lab.sh" 3.2
