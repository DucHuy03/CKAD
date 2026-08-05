#!/usr/bin/env bash
# Removes only the Lab 5.4 Helm release.
set -euo pipefail
helm uninstall "${RELEASE:-lab54-movie-service}" -n "${NAMESPACE:-lab}" --ignore-not-found
