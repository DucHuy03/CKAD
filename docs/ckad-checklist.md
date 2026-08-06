# CKAD checklist and current evidence

`scripts/check-capstone.sh` is a static audit. Items marked **partial** or
**missing** are known, deliberately deferred gaps — not oversights.

| ID | Status | Current evidence / gap |
| --- | --- | --- |
| D1 | done | Five Dockerfiles exist; core service manifests use `1.0.0` tags and `scripts/build.sh`. |
| D2 | done | `movie-booking/k8s/booking-service/cronjob.yaml` — production `booking-cleanup-cronjob` (every 5 min, real `-cleanup-expired-holds` mode, reuses the real ConfigMap/Secret). The one-off migration Job is also demoed in `labs/lab-1.3`; the main deploy path covers the same migration via each service's `init-db-migrate` init container instead of a separate Job. |
| D3–D4 | done | Init container, log-shipper sidecar, nginx-ambassador sidecar and `emptyDir` in every core service Deployment. |
| D5 | done | Postgres StatefulSet volume claims (×4) and RabbitMQ PVC; data survives Pod delete/recreate. |
| D6 | partial | `app` labels consistent everywhere; blue/green labels exist only in `labs/lab-2.2`, not normalised into the main app. |
| P1–P2 | done | Five long-running Deployments (≥1 replica each); rolling update documented and demoed via `labs/lab-2.1` and `docs/demo-script.md`. |
| P3 | partial | Blue/green (Service selector flip) demoed in `labs/lab-2.2`; not present as a second Deployment in the main app manifests. |
| P4 | done | `k8s/autoscaling/booking-service-hpa.yaml` (CPU target on booking-service). |
| P5 | partial | Kustomize `base/` + `overlays/{dev,prod}` exists for `movie-service` only; other four services are plain manifests. |
| P6 | done | `helm/movie-service` is a single generic chart; `values.yaml` holds one entry per core service (`services.<name>`) with shared `&defaults` anchor, selected via `--set serviceName=<svc>`. Install/upgrade/rollback verified live via `labs/lab-5.4`. |
| C1 | done | Per-service ConfigMap injected via `envFrom` in every core service Deployment. |
| C2 | partial | `scripts/create-secrets.sh` / `generate-local-secrets.sh` produce runtime Secrets; the real `postgres123`/RabbitMQ passwords are still present in early git history (no rewrite planned — only the live password will be rotated). |
| C3 | done | `allowPrivilegeEscalation: false`, `capabilities.drop: [ALL]`, `readOnlyRootFilesystem: true` on the `app` container of **all five** core services (previously gateway-only). |
| C4 | done | `booking-observer` ServiceAccount + Role + RoleBinding (get/list Pods only), bound and used by the `booking-service` Deployment. |
| C5 | done | `k8s/quota/` — ResourceQuota + LimitRange on `movie-booking` (and a separate `lab-defaults` LimitRange on the `lab` namespace). |
| C6 | done | Every container checked (app/init/sidecars, all 4 Postgres, Redis, RabbitMQ, MailHog) declares CPU/memory `requests` and `limits`. |
| N1–N2 | done | Internal ClusterIP Services for east-west traffic; NodePort `api-gateway` (30080) + Ingress for external entry. |
| N3 | done | Ingress routes ≥2 paths (`/api`, `/movies`) to different backends. |
| N4 | done* | `default-deny-ingress` now covers every Pod in the namespace (not just the 5 API Pods); explicit allow-lists added per datastore (each Postgres from its one owning service, Redis from booking-service, RabbitMQ AMQP from payment+notification-service, MailHog SMTP from notification-service) plus the existing gateway/ingress/egress rules. *Manifests are complete and apply cleanly, but Docker Desktop ships no policy-capable CNI, so enforcement can't be demonstrated live in this environment (documented in README "Known limitations"). |
| N5 | partial | `movie-booking/k8s/phase5-testing/{smoke-test.sh,e2e-test.sh}` check Endpoints; still assume the `docker-desktop` context specifically. |
| O1–O2 | done* | Liveness+readiness on all 5 core services, all 4 Postgres, Redis, and RabbitMQ. *MailHog (a demo-only SMTP catcher, no HTTP health contract of its own beyond the web UI) has resources but no probes — minor, undemonstrated gap, not a core service. |
| O3 | done | RabbitMQ has a `startupProbe` (~150s ceiling) ahead of its readiness/liveness probes, for its slow Erlang VM boot. |
| O4 | done | README "Debugging" section covers `kubectl logs` (incl. `-c`/`--previous`), `describe`, `get events --sort-by`, and `top`, plus the Endpoints/selector-mismatch triage tip. |
| O5 | done | All workloads use current stable APIs (`apps/v1`, `networking.k8s.io/v1`, etc.); no deprecated Ingress/extensions API in use. |

## Still open (deferred by choice, not urgent)

These are known and intentionally not yet done — see the corresponding row
above for why each one is lower priority than what's already fixed:

1. **P3** — blue/green (Service selector flip) exists and works in
   `labs/lab-2.2`, but isn't promoted into the main `movie-booking/k8s`
   deploy path as a second Deployment. Bigger lift than the CronJob above
   (needs a real second image/replica set wired to a selector-flip demo).
2. **P5** — Kustomize base/overlay only wraps `movie-service`; extending to
   all five services means writing 4 more bases.
3. **C2** — the training `postgres123` password is still visible in early
   git history on the public repo. Per explicit instruction, only the live
   in-cluster password will be rotated (once `kubectl exec` is reliable
   again); no history rewrite is planned.
4. **N5** — smoke/e2e scripts hard-assume `docker-desktop`; making them
   context-agnostic is a small script change, not yet done.
