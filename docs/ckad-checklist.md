# CKAD checklist and current evidence

`scripts/check-capstone.sh` is a static audit. Items marked **partial** or
**missing** are intentional Phase 2 work, not claims of compliance.

| ID | Status | Current evidence / gap |
| --- | --- | --- |
| D1 | done | Five Dockerfiles exist; core service manifests use `1.0.0` tags and `scripts/build.sh`. |
| D2 | partial | Deployments and lab Job/CronJob exist; production Job/CronJob is not packaged. |
| D3–D4 | done | Init, log sidecar, ambassador and `emptyDir` in service deployments. |
| D5 | done | Postgres StatefulSet claims and RabbitMQ PVC. |
| D6 | partial | `app` labels and blue/green lab exist; label convention needs normalising. |
| P1–P3 | partial | Five Deployments and blue/green manifests exist; rollout demo needs stable image tags. |
| P4 | done | `k8s/autoscaling/booking-service-hpa.yaml`. |
| P5 | partial | Kustomize base/dev/prod exists for movie service only. |
| P6 | partial | `helm/movie-service` provides installable service packaging; verify upgrade/rollback on cluster. |
| C1 | done | ConfigMaps are injected into service Pods. |
| C2 | partial | `scripts/create-secrets.sh` creates runtime Secrets; lab fixtures and Docker Compose training defaults still need final secret cleanup. |
| C3–C4 | partial | Gateway has a production SecurityContext; least-privilege Role/ServiceAccount manifest exists. |
| C5 | done | `k8s/quota/` contains ResourceQuota and LimitRange. |
| C6 | partial | Core application init/app/sidecar containers have resources; audit infrastructure workloads before submission. |
| N1–N2 | done | Internal ClusterIP Services and NodePort api-gateway exist. |
| N3–N4 | partial | Ingress and ingress-isolation policies added; expand policy for all required application flows before enforcing in production. |
| N5 | partial | Smoke test checks endpoints; make it environment-independent. |
| O1–O2 | partial | Core API containers have probes; audit all infrastructure deployments. |
| O3 | missing | No startup probe or documented rationale. |
| O4 | partial | Infrastructure/debug notes exist; root runbook is incomplete. |
| O5 | done | Existing workload APIs use stable API versions. |

## Submission blockers

1. Remove real/plaintext credentials from Git and generate runtime Secrets.
2. Replace all `:latest` references with immutable release tags or digests.
3. Implement P4, P6, N3 and N4.
