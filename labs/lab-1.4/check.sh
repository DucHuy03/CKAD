#!/usr/bin/env bash
# Stable, read-only checker for Lab 1.4; it verifies label-drill assets.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/check-lab.sh" 1.4
