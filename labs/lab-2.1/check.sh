#!/usr/bin/env bash
# Stable, read-only checker for Lab 2.1; rollout execution is in run.sh.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/check-lab.sh" 2.1
