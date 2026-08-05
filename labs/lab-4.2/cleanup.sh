#!/usr/bin/env bash
# Removes only the Lab 4.2 Ingress resource.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts/cleanup-lab-manifests.sh" 4.2
