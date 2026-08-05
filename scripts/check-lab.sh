#!/usr/bin/env bash
# Read-only evidence check for one CKAD lab.
# Usage: bash scripts/check-lab.sh 1.1
# This checker deliberately does not apply/delete Kubernetes resources; run.sh
# is for the hands-on exercise, while check.sh is safe for CI and review.

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/check.sh"
LAB_ID="${1:-}"

if [[ ! "$LAB_ID" =~ ^[1-5]\.[1-4]$ ]]; then
  echo "Usage: $0 <day.lab>, for example: $0 1.1" >&2
  exit 2
fi

require_file "Lab/${LAB_ID//./_}.md" "Lab $LAB_ID instructions exist"

case "$LAB_ID" in
  1.1|1.2|1.3|1.4|2.1|2.2|2.3|2.4|3.1|3.2|3.3|3.4) require_file "labs/lab-$LAB_ID/run.sh" "Lab runner exists" ;;
  4.1) require_file "labs/lab-4.1/run.sh" "Lab runner exists"; require_file "labs/lab-4.1/manifests/backend-deployment.yaml" "ClusterIP and NodePort manifests exist" ;;
  4.2) require_file "labs/lab-4.2/run.sh" "Lab runner exists"; require_file "labs/lab-4.2/manifests/path-routing-ingress.yaml" "Ingress manifest exists" ;;
  4.3) require_file "labs/lab-4.3/run.sh" "Lab runner exists"; require_file "labs/lab-4.3/manifests/network-isolation.yaml" "NetworkPolicy manifest exists" ;;
  4.4) require_file "labs/lab-4.4/run.sh" "Lab runner exists"; require_file "labs/lab-4.4/manifests/persistent-volume-claim.yaml" "PVC manifest exists" ;;
  5.1) require_file "labs/lab-5.1/run.sh" "Lab runner exists"; require_file "labs/lab-5.1/manifests/self-healing-deployment.yaml" "Probe Deployment exists" ;;
  5.2) require_file "labs/lab-5.2/run.sh" "CLI observability runner exists"; require_file "labs/lab-5.1/manifests/self-healing-deployment.yaml" "Observability target exists" ;;
  5.3) require_file "labs/lab-5.3/run.sh" "Lab runner exists"; require_file "labs/lab-5.3/manifests/broken-service.yaml" "Broken fixture exists"; require_file "labs/lab-5.3/manifests/fixed-deployment.yaml" "Fixed fixture exists" ;;
  5.4) require_file "labs/lab-5.4/run.sh" "Helm runner exists"; require_file "helm/movie-service/Chart.yaml" "Helm chart exists" ;;
esac

finish
