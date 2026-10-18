# Backlog / lag do worker (CrosswatcWorkerBacklog / CrosswatcReconcilerLag / CrosswatcWorkerDown)

1. Só o líder processa: `kubectl get lease crosswatc-worker -o yaml` — o holder está vivo?
2. `crosswatc_reconcile_duration_seconds` alto → GitHub lento (rate limit do App) ou API server lento; veja spans `reconciler.resource`.
3. `SELECT count(*) FROM inbox WHERE processed_at IS NULL` cresce → worker parado ou erro permanente (`last_error`).
