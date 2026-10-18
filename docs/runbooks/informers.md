# Informers desconectando (CrosswatcInformerDisconnected)

1. `kubectl logs deploy/crosswatc-worker | grep 'informer watch error'` — o label `watch` indica o GVR.
2. `403` → RBAC: o grupo do Claim/MR não está em `rbac.platformApiGroups`/`providerApiGroups`.
3. `410 Gone`/timeouts frequentes → API server sobrecarregado; o informer relista sozinho.
