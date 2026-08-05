#!/usr/bin/env bash
# Stable, read-only checker for Lab 1.3; it verifies Job and CronJob evidence.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/check-lab.sh" 1.3
