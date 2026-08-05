#!/usr/bin/env bash
# Builds every local image with an explicit version tag used by Kubernetes.
# Set IMAGE_TAG=1.0.1 to produce a new release without editing manifests.

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="$SCRIPT_DIR/../movie-booking"
IMAGE_TAG="${IMAGE_TAG:-1.0.0}"

for service in movie-service booking-service payment-service notification-service api-gateway; do
  docker build -t "$service:$IMAGE_TAG" "$APP_ROOT/$service"
done
docker build -t "log-shipper:$IMAGE_TAG" "$APP_ROOT/shared-images/log-shipper"
docker build -t "nginx-ambassador:$IMAGE_TAG" "$APP_ROOT/shared-images/nginx-ambassador"
