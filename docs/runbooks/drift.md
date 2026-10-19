# Pico de drift (CrosswatcDriftSpike)

1. Liste recursos com drift: `GET /api/v1/resources` + `/state` (`drift.detected`), ou o banner na UI.
2. `git-claim-mismatch` em massa → alguém aplicou manifests fora do Git, ou o Argo está pausado.
3. `cloud-reconcile-error`/`mr-not-ready` em massa → incidente no provider de cloud ou credenciais do ProviderConfig expiradas.
