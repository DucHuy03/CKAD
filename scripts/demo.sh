#!/usr/bin/env bash
# Read-only readiness check for the capstone live demo (docs/demo-script.md).
# Verifies the precondition for each capstone-requirements.md §6.3 item is
# currently true on the live cluster. It never creates, deletes, or patches
# anything — the demo itself (rollout, Pod delete, etc.) happens live, by
# hand, following docs/demo-script.md, not inside this script.
set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/check.sh"

NAMESPACE="movie-booking"
CORE_APPS=(api-gateway movie-service booking-service payment-service notification-service)

echo "=== 1. All Pods Running/Ready; Services have Endpoints ==="
for app in "${CORE_APPS[@]}"; do
  ready="$(kubectl get pods -n "$NAMESPACE" -l "app=$app" -o jsonpath='{.items[0].status.containerStatuses[?(@.ready==true)].name}' 2>/dev/null | wc -w)"
  total="$(kubectl get pods -n "$NAMESPACE" -l "app=$app" -o jsonpath='{.items[0].spec.containers[*].name}' 2>/dev/null | wc -w)"
  if [[ "$total" -gt 0 && "$ready" == "$total" ]]; then
    pass "$app Pod is $ready/$total Ready"
  else
    fail "$app Pod is $ready/$total Ready"
  fi

  endpoints="$(kubectl get endpoints "$app" -n "$NAMESPACE" -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null)"
  [[ -n "$endpoints" ]] && pass "$app Service has Endpoints ($endpoints)" || fail "$app Service has no Endpoints"
done

echo
echo "=== 2. External exposure: NodePort + Ingress object ==="
if curl --max-time 5 -sf http://localhost:30080/healthz >/dev/null 2>&1; then
  pass "NodePort gateway (localhost:30080) responds to /healthz"
else
  fail "NodePort gateway (localhost:30080) did not respond to /healthz"
fi
if kubectl get ingress ticket-booking -n "$NAMESPACE" >/dev/null 2>&1; then
  pass "Ingress object 'ticket-booking' exists"
else
  fail "Ingress object 'ticket-booking' exists"
fi
if kubectl get ingressclass nginx >/dev/null 2>&1; then
  pass "An nginx IngressClass/controller is registered"
else
  todo "No ingress controller registered yet — Ingress object exists but won't route traffic; demo the NodePort path instead"
fi

echo
echo "=== 3. ConfigMap/Secret injection ==="
for app in "${CORE_APPS[@]}"; do
  refs="$(kubectl get deployment "$app" -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].envFrom[*].configMapRef.name}{" "}{.spec.template.spec.containers[0].envFrom[*].secretRef.name}' 2>/dev/null)"
  if [[ "$refs" == *"config"* && "$refs" == *"secret"* ]]; then
    pass "$app app container has ConfigMap + Secret envFrom ($refs)"
  else
    fail "$app app container is missing ConfigMap and/or Secret envFrom ($refs)"
  fi
done

echo
echo "=== 4. Probes configured (liveness + readiness) ==="
for app in "${CORE_APPS[@]}"; do
  live="$(kubectl get deployment "$app" -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].livenessProbe}' 2>/dev/null)"
  ready="$(kubectl get deployment "$app" -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].readinessProbe}' 2>/dev/null)"
  if [[ -n "$live" && -n "$ready" ]]; then
    pass "$app app container has liveness + readiness probes"
  else
    fail "$app app container is missing a liveness and/or readiness probe"
  fi
done

echo
echo "=== 5. Rolling update / rollback evidence ==="
revisions="$(kubectl rollout history deployment/movie-service -n "$NAMESPACE" 2>/dev/null | grep -c '^[0-9]')"
if [[ "$revisions" -ge 1 ]]; then
  pass "movie-service has rollout history ($revisions revision(s)) — demo live with: bash labs/lab-2.1/run.sh"
else
  todo "movie-service has no rollout history yet"
fi

echo
echo "=== 6. HPA present ==="
hpa_count="$(kubectl get hpa -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)"
[[ "$hpa_count" -ge 1 ]] && pass "$hpa_count HPA object(s) present" || fail "No HPA object present"

echo
echo "=== 7. NetworkPolicy present ==="
netpol_count="$(kubectl get networkpolicy -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)"
if [[ "$netpol_count" -ge 1 ]]; then
  pass "$netpol_count NetworkPolicy object(s) present"
  kubectl get networkpolicy -n "$NAMESPACE" -o custom-columns=NAME:.metadata.name,PODSELECTOR:.spec.podSelector 2>/dev/null | tail -n +2 | sed 's/^/       /'
else
  fail "No NetworkPolicy object present"
fi
# This only checks the object exists; it deliberately does not test
# enforcement (that would mean creating a throwaway Pod, which this
# read-only script never does). Docker Desktop's built-in Kubernetes has no
# policy-capable CNI (no Calico/Cilium/Weave in kube-system), so these
# objects are currently NOT enforced. See docs/demo-script.md step 7.
if ! kubectl get pods -n kube-system 2>/dev/null | grep -qiE "calico|cilium|weave|antrea"; then
  todo "No policy-capable CNI detected in kube-system — NetworkPolicy objects exist but traffic is not actually being blocked on this cluster"
fi

echo
echo "=== 8. PVC bound (persistence) ==="
pvc_count="$(kubectl get pvc -n "$NAMESPACE" --no-headers 2>/dev/null | grep -c Bound)"
[[ "$pvc_count" -ge 1 ]] && pass "$pvc_count PVC(s) Bound" || fail "No PVC is Bound"

echo
echo "=== 9. Kustomize / Helm packaging ==="
if kubectl kustomize "$REPO_ROOT/movie-booking/k8s-kustomize/overlays/dev" >/dev/null 2>&1; then
  pass "Kustomize dev overlay builds cleanly — demo live with: bash labs/lab-2.4/run.sh"
else
  fail "Kustomize dev overlay does not build"
fi
if command -v helm >/dev/null 2>&1; then
  pass "helm CLI is installed — demo live with: bash labs/lab-5.4/run.sh"
else
  todo "helm CLI is not installed on this machine; use the Kustomize overlay demo instead"
fi

finish
