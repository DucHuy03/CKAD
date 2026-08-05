#!/usr/bin/env bash
# Stable checker for Lab 5.3; broken-YAML triage assets are planned for Phase 2.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/check-lab.sh" 5.3
