# crosswatc Helm chart

Single-cluster install of `crosswatc-web`, `crosswatc-api` and `crosswatc-worker` (spec §7, §26, §27).

```bash
kubectl create secret generic crosswatc-db --from-literal=DATABASE_URL='postgres://...'
kubectl create secret generic crosswatc-github-app --from-file=private-key.pem
helm install crosswatc deploy/helm/crosswatc \
  --set auth.oidcIssuer=https://idp.example.com \
  --set github.enabled=true --set github.appID=123 --set github.installationID=456 \
  --set 'rbac.platformApiGroups={database.crosswatc.io,compute.crosswatc.io}' \
  --set 'rbac.providerApiGroups={dbforpostgresql.azure.upbound.io,network.azure.upbound.io}'
```

## RBAC

- `ClusterRole` is **read-only** (`get`, `list`, `watch`): XRDs, Compositions, CompositionRevisions, MRDs, CRDs,
  Argo CD `Application`s, Events, and the API groups listed in `rbac.platformApiGroups` / `rbac.providerApiGroups`.
- `rbac.readAllGroups=true` grants read on every group — simpler for a pilot, broader than needed.
- `secrets: get, list, watch` is required to check connection Secret existence and to run the metadata-only Secret
  informer. The code only calls the metadata API and never reads values (§49), but Kubernetes RBAC cannot restrict
  verbs to metadata. Set `rbac.secretMetadata=false` to drop it (the Secret informer is disabled and the Operational
  stage will report the secret as unavailable).
- The only write permission is a namespaced `Role` on `coordination.k8s.io/leases` for leader election (§35.2).

## Webhooks (spec §35.1)

```bash
kubectl create secret generic crosswatc-webhooks \
  --from-literal=github-webhook-secret=... --from-literal=argocd-webhook-token=...
helm upgrade crosswatc deploy/helm/crosswatc --reuse-values --set webhooks.existingSecret=crosswatc-webhooks
```

- GitHub App → `POST https://<host>/webhooks/github` (events: pull request, check run, pull request review, push).
- argocd-notifications webhook service → `POST https://<host>/webhooks/argocd` with `Authorization: Bearer <token>`
  and the body documented in `internal/events` (`eventId`, `app`, `namespace`, `event`, `revision`, `timestamp`);
  triggers `on-sync-running`, `on-sync-succeeded`, `on-sync-failed`, `on-health-degraded`.
- Deliveries are queued in the `inbox` table (idempotent by delivery id) and consumed by the leader worker,
  which re-reads GitHub / the Argo `Application` before reconciling. The periodic resync covers lost webhooks.

## Notes

- `worker.replicas: 2` with leader election on the `crosswatc-worker` Lease (§35.2). Only the leader runs informers,
  polling and the reconciler; the standby is ready as soon as the database answers and takes over within
  `leaseDuration` (immediately on graceful shutdown, since the Lease is released).
- Set `worker.leaderElection.enabled=false` only with `worker.replicas: 1`.
- The worker owns database migrations; the API only reads.
- `auth.mode=dev` is rejected by the API when `env=production`.
