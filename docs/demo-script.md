# Live demo script (5–10 minutes)

Walks through `capstone-requirements.md` §6.3 in order, against the real
`movie-booking` namespace. This is **not** one of the 20 CKAD labs — those
are individual skill drills; this is the single presentation for the
instructor. Before presenting, run the read-only precondition check:

```bash
bash scripts/demo.sh
```

Any `[FAIL]` line there is a real gap to fix before you present — the steps
below assume every core service is `Ready`.

---

## 1. Pods Running/Ready; Services have Endpoints (~1 min)

```bash
kubectl get pods -n movie-booking
kubectl get endpoints -n movie-booking
```
Say: every core service Pod is `3/3` (`app` + `log-shipper` +
`nginx-ambassador`, plus an init container that only shows up while it's
still migrating), and every Service has at least one Pod IP under
`ENDPOINTS` — an empty list there would mean the Service's selector doesn't
match any Ready Pod.

## 2. External exposure hits API paths (~1 min)

```bash
curl http://localhost:30080/healthz
curl http://localhost:30080/api/movies
```
Say: `api-gateway` is reachable over NodePort `30080` and proxies
`/api/movies` to `movie-service`. An `Ingress` object (`ticket-booking`,
host `ticket-booking.local`, paths `/api` and `/movies`) is also defined —
if `kubectl get ingressclass` shows no controller registered in this
environment, say so plainly and fall back to the NodePort path rather than
pretending it routes.

## 3. ConfigMap/Secret injection (~1 min)

```bash
kubectl describe deployment movie-service -n movie-booking | grep -A4 "Environment Variables"
kubectl exec deploy/movie-service -n movie-booking -c app -- env | grep -E "DB_HOST|DB_PASSWORD"
```
Say: `DB_HOST`/`DB_PORT`/`DB_NAME` come from `movie-service-config`
(ConfigMap), `DB_PASSWORD` from `movie-service-secret` (Secret) — same
`envFrom` pattern on all five services. If `kubectl exec` isn't available in
this environment, the `describe` output alone (envFrom source names) is
still evidence; skip the second command.

## 4. Probe behavior (~1 min)

```bash
kubectl describe pod -n movie-booking -l app=movie-service | grep -E "Liveness|Readiness"
```
Say: `/healthz` (liveness) and `/ready` (readiness) are both HTTP probes on
the `app` container; `nginx-ambassador` has its own pair on
`/ambassador-healthz`. Optional live break: `kubectl scale deployment
postgres-movie --replicas=0 -n movie-booking` briefly makes `/ready` start
failing DB checks and Endpoints drop — remember to scale it back to `1`
right after.

## 5. Rolling update / rollback (~2 min)

```bash
bash labs/lab-2.1/run.sh
```
Say: this builds `movie-service:v1`/`:v2`, rolls out v1→v2 with `kubectl
rollout status`, deliberately deploys a non-existent image tag to force a
failed rollout, then `kubectl rollout undo`s it — and restores the original
Deployment automatically on exit (`trap`), so it's safe to run live.

## 6. HPA object present (~30 sec)

```bash
kubectl get hpa -n movie-booking
```
Say: `booking-service` autoscales 1–5 replicas on 50% CPU target
(`movie-booking/k8s/autoscaling/booking-service-hpa.yaml`).

## 7. NetworkPolicy effect: allowed vs denied path (~2 min)

**Check this before you present, not during:** `kubectl get pods -n
kube-system` needs a policy-capable CNI (Calico/Cilium/Weave/Antrea) for
these objects to actually block anything. Docker Desktop's built-in
Kubernetes does **not** ship one — on that cluster the command below will
show the "denied" Pod reaching `movie-service` anyway, which would
contradict what you're saying live. Run it once beforehand:

```bash
kubectl run netpol-denied --image=busybox --restart=Never -n movie-booking \
  --labels="app=netpol-denied" -- sh -c 'wget -T 3 -qO- http://movie-service:8080/healthz || echo BLOCKED'
sleep 4 && kubectl logs netpol-denied -n movie-booking
kubectl delete pod netpol-denied -n movie-booking
```
If it prints `{"status":"alive"}` instead of `BLOCKED`, do **not** run this
step live — present the NetworkPolicy YAML and explain the intended
behavior instead, and say plainly that this environment's CNI doesn't
enforce it (true and defensible; faking a passing demo is worse). On a
policy-capable cluster (a real cloud cluster, or Calico installed), this
step will correctly print `BLOCKED`.

For the allowed path once enforcement does work: booking-service (in the
allow-list) reaching movie-service is exactly what already happens on every
real booking request — point back at the `/api/movies` response from step 2
as that evidence.

Say (only if enforcement is confirmed working): `default-deny-ingress` blocks all ingress to the five app Pods by
default; `allow-gateway-to-apis` opens port 9090 only from
`api-gateway`/`booking-service`/`payment-service`; `restrict-egress` caps
outbound traffic to DNS and same-namespace only, denying the open internet.
This uses `kubectl logs` deliberately instead of `kubectl exec`, since exec
needs a working streaming proxy that isn't always available in every
environment.

## 8. PVC persistence after Pod recreate (~2 min)

```bash
curl -s http://localhost:30080/api/movies | head -c 300; echo
kubectl delete pod postgres-movie-0 -n movie-booking
kubectl wait --for=condition=Ready pod/postgres-movie-0 -n movie-booking --timeout=60s
curl -s http://localhost:30080/api/movies | head -c 300; echo
```
Say: `postgres-movie-0` is a StatefulSet Pod backed by PVC
`data-postgres-movie-0`; deleting the Pod only deletes the compute, the
`/api/movies` response is identical before and after because the data lives
on the PVC, not in the Pod.

## 9. Kustomize overlay apply (~1–2 min)

```bash
bash labs/lab-2.4/run.sh
```
Say: `movie-booking/k8s-kustomize/base` is one set of manifests; `overlays/dev`
(1 replica, `v1`) and `overlays/prod` (3 replicas, `v2`) patch only what
differs via `namePrefix`/`images`/`replicas` transformers — no manifest is
duplicated. This also self-restores on exit. (If `helm` is installed, `bash
labs/lab-5.4/run.sh` demonstrates the Helm install/upgrade/rollback
alternative instead — see `helm/movie-service/`.)

---

## If something is red right before you present

Run `bash scripts/demo.sh` first. Each `[FAIL]` line names the exact Service
or Deployment to `kubectl describe`; see the "Debugging" section in the root
[`README.md`](../README.md) for the standard triage order.
