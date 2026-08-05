#!/usr/bin/env bash
# Applies the Lab 4.2 Ingress; run Lab 4.1 first to create its backend Services.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/run-lab-manifests.sh" 4.2
