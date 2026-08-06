#!/usr/bin/env bash
# Installs, upgrades, shows history, then rolls back the local movie-service chart.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RELEASE="${RELEASE:-lab54-movie-service}"
NAMESPACE="${NAMESPACE:-lab}"

# The chart intentionally ships no configmap.yaml template (envFrom references
# a plain "movie-service-config" by name, same external-dependency pattern as
# the Secret) - pre-create it here so the release is self-contained. DB_HOST
# is the FQDN, not the bare Service name: this lands in "lab", a different
# namespace than where postgres-movie actually runs (movie-booking).
kubectl create configmap movie-service-config -n "$NAMESPACE" \
  --from-literal=DB_HOST=postgres-movie.movie-booking.svc.cluster.local \
  --from-literal=DB_PORT=5432 \
  --from-literal=DB_NAME=movie_service \
  --from-literal=DB_SSLMODE=disable \
  --from-literal=HTTP_PORT=8080 \
  --from-literal=LOG_FILE_PATH=/var/log/app/app.log \
  --dry-run=client -o yaml | kubectl apply -f -

# image.tag=1.0.1 must exist locally before the upgrade step below pulls it
# (imagePullPolicy: IfNotPresent, no registry push in this project).
if ! docker image inspect movie-service:1.0.1 >/dev/null 2>&1; then
  docker build -t movie-service:1.0.1 "$ROOT/movie-booking/movie-service"
fi

# The chart's fullname (templates/_helpers.tpl) is Values.serviceName, which
# defaults to "movie-service" (values.yaml) - the Deployment/Service it
# creates are always called "movie-service" in this namespace, regardless of
# $RELEASE, unless --set serviceName=<other-service> is passed.
WORKLOAD="movie-service"

helm upgrade --install "$RELEASE" "$ROOT/helm/movie-service" -n "$NAMESPACE" --create-namespace --set services.movie-service.image.tag=1.0.0
kubectl rollout status "deployment/$WORKLOAD" -n "$NAMESPACE" --timeout=60s
helm upgrade "$RELEASE" "$ROOT/helm/movie-service" -n "$NAMESPACE" --set services.movie-service.image.tag=1.0.1
kubectl rollout status "deployment/$WORKLOAD" -n "$NAMESPACE" --timeout=60s
helm history "$RELEASE" -n "$NAMESPACE"
helm rollback "$RELEASE" 1 -n "$NAMESPACE"
kubectl rollout status "deployment/$WORKLOAD" -n "$NAMESPACE" --timeout=60s
