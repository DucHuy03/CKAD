#!/usr/bin/env bash
# Stable, read-only checker for Lab 3.4; it verifies quota exercise assets.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/check-lab.sh" 3.4
