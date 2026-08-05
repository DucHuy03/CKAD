#!/usr/bin/env bash
# Shared helpers for read-only lab and capstone checks.
# Every caller obtains REPO_ROOT from this file, avoiding fragile `cd` usage.

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PASS=0
FAIL=0
TODO=0

pass() { printf '[PASS] %s\n' "$1"; PASS=$((PASS + 1)); }
fail() { printf '[FAIL] %s\n' "$1"; FAIL=$((FAIL + 1)); }
todo() { printf '[TODO] %s\n' "$1"; TODO=$((TODO + 1)); }

# require_file checks repository evidence without changing files or the cluster.
require_file() {
  local path="$1" description="$2"
  [[ -f "$REPO_ROOT/$path" ]] && pass "$description" || fail "$description ($path)"
}

# finish prints a uniform summary. TODO is informational; FAIL controls exit code.
finish() {
  printf '\nSummary: %s pass, %s fail, %s todo\n' "$PASS" "$FAIL" "$TODO"
  return "$FAIL"
}
