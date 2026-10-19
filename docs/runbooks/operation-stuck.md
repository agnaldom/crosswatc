# Operação sem progresso (CrosswatcOperationStuck)

Evento `OPERATION.STUCK` na timeline do recurso indica o estado parado.

| Estado | Quem age |
|---|---|
| `VALIDATING` / `WAITING_APPROVAL` | Solicitante / aprovador no PR |
| `MERGED` / `SYNCING` | Plataforma: Argo CD Application (`argocd app get`) |
| `RECONCILING` / `APPLYING` | Plataforma: `kubectl describe` no XR/MR; condition `Synced=False` com `ReconcileError` |

Após `CROSSWATC_STUCK_TIMEOUT` com erro de reconciliação a revision vira `FAILED`.
