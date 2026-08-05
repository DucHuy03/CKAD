#!/usr/bin/env bash
# Creates the isolated Lab credential Secret; the password is never committed.
set -euo pipefail
NAMESPACE="${NAMESPACE:-lab}"
: "${LAB_DB_PASSWORD:?Set LAB_DB_PASSWORD before running labs that use a database}"
kubectl create secret generic lab-db-credentials -n "$NAMESPACE" \
  --from-literal=DB_PASSWORD="$LAB_DB_PASSWORD" --dry-run=client -o yaml | kubectl apply -f -

# Lab 2.4 (Kustomize) deploys movie-service into "lab" from
# movie-booking/k8s-kustomize/base/movie-service, whose Deployment references
# a plain "movie-service-secret" via secretRef. Kustomize's namePrefix only
# rewrites references to resources it manages; since Secrets are never
# committed to git, this one is external to the kustomization and its name
# stays unprefixed in the rendered output for both the dev and prod overlays.
kubectl create secret generic movie-service-secret -n "$NAMESPACE" \
  --from-literal=DB_USER=postgres --from-literal=DB_PASSWORD="$LAB_DB_PASSWORD" \
  --dry-run=client -o yaml | kubectl apply -f -
