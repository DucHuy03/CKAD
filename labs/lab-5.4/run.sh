#!/usr/bin/env bash
# Installs, upgrades, shows history, then rolls back the local movie-service chart.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RELEASE="${RELEASE:-lab54-movie-service}"
NAMESPACE="${NAMESPACE:-lab}"
helm upgrade --install "$RELEASE" "$ROOT/helm/movie-service" -n "$NAMESPACE" --create-namespace --set image.tag=1.0.0
helm upgrade "$RELEASE" "$ROOT/helm/movie-service" -n "$NAMESPACE" --set image.tag=1.0.1
helm history "$RELEASE" -n "$NAMESPACE"
helm rollback "$RELEASE" 1 -n "$NAMESPACE"
