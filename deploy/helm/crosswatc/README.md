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
- `secrets: get` is required to check connection Secret existence. The code only calls the metadata API and never
  reads values (§49), but Kubernetes RBAC cannot restrict `get` to metadata. Set `rbac.secretMetadata=false` to drop it
  (the Operational stage will then report the secret as unavailable).
- The only write permission is a namespaced `Role` on `coordination.k8s.io/leases` for leader election (§35.2).

## Notes

- `worker.replicas` stays at 1 until leader election ships (issue P2.1).
- The worker owns database migrations; the API only reads.
- `auth.mode=dev` is rejected by the API when `env=production`.
