#!/usr/bin/env bash
# Stable, read-only checker for Lab 2.3; it records the remaining HPA manifest work.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/check-lab.sh" 2.3
