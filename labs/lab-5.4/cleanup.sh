#!/usr/bin/env bash
# Removes the Lab 5.4 Helm release and the ConfigMap run.sh pre-created for it.
set -euo pipefail
NAMESPACE="${NAMESPACE:-lab}"
helm uninstall "${RELEASE:-lab54-movie-service}" -n "$NAMESPACE" --ignore-not-found
kubectl delete configmap movie-service-config -n "$NAMESPACE" --ignore-not-found
