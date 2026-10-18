# API availability (CrosswatcAPIAvailabilityBudgetBurn)

SLO: 99.5% das requests da API sem 5xx (spec §53).

1. `kubectl logs deploy/crosswatc-api | grep '"status":5'` — agrupe por `path` e `correlation.id`.
2. `/readyz` falhando → PostgreSQL indisponível ou pool esgotado: verifique conexões e o secret `crosswatc-db`.
3. 502 em `/changes` → GitHub indisponível ou GitHub App sem permissão (ver `webhooks.md`).
4. Se for regressão de deploy, faça rollback do chart (`helm rollback crosswatc`).
