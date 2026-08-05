#!/usr/bin/env bash
# Stable, read-only checker for Lab 1.2; it delegates common logic to scripts/check-lab.sh.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/check-lab.sh" 1.2
