#!/usr/bin/env bash
# Stable, read-only checker for Lab 2.4; it verifies both Kustomize overlays.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/check-lab.sh" 2.4
