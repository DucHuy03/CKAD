#!/usr/bin/env bash
# Static Phase 1 audit for the CKAD capstone.
# It only reads tracked project files. Cluster checks will be added separately
# because they require a configured kubecontext and installed cluster features.

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/check.sh"

MANIFEST_ROOT="$REPO_ROOT/movie-booking/k8s"

require_file "README.md" "Root README documents the repository"
require_file "docs/architecture.md" "Architecture documentation exists"
require_file "docs/ckad-checklist.md" "CKAD evidence checklist exists"

for service in movie-service booking-service payment-service notification-service api-gateway; do
  require_file "movie-booking/$service/Dockerfile" "$service has a Dockerfile"
  require_file "movie-booking/k8s/$service/deployment.yaml" "$service has a Deployment"
done

# Explicit tags are mandatory for the final submission. grep is used instead
# of ripgrep so this checker also runs in a default Git Bash installation.
if grep -Rqs --include='*.yaml' --include='*.yml' 'image: .*:latest' "$MANIFEST_ROOT"; then
  fail "No Kubernetes image uses :latest"
else
  pass "No Kubernetes image uses :latest"
fi

if grep -Rqs --include='*.yaml' --include='*.yml' 'kind: Ingress' "$MANIFEST_ROOT"; then pass "Ingress manifest exists"; else fail "Ingress manifest exists"; fi
if grep -Rqs --include='*.yaml' --include='*.yml' 'kind: NetworkPolicy' "$MANIFEST_ROOT"; then pass "NetworkPolicy manifest exists"; else fail "NetworkPolicy manifest exists"; fi
if grep -Rqs --include='*.yaml' --include='*.yml' 'kind: HorizontalPodAutoscaler' "$MANIFEST_ROOT"; then pass "HPA manifest exists"; else fail "HPA manifest exists"; fi
if [[ -d "$REPO_ROOT/helm" ]]; then pass "Helm chart directory exists"; else fail "Helm chart directory exists"; fi

# Flag only likely training credentials, so the message is actionable without
# attempting to decode or print secret data.
if grep -RqsE --include='*.yaml' --include='*.yml' 'postgres123|dev-only-insecure-secret|k8s-lab-secret' "$REPO_ROOT/movie-booking"; then
  fail "No training/plaintext credentials are stored in manifests"
else
  pass "No training/plaintext credentials are stored in manifests"
fi

finish
