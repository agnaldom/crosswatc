# ADR-005 — Observação via informers

- **Status:** Aceito
- **Data:** 2026-10-17
- **Spec:** §25, §35.2

## Contexto

Polling periódico do cluster inteiro é caro e atrasa a UI; a meta é evento externo → UI em menos de 3 s.

## Decisão

O worker líder (Lease `crosswatc-worker`) mantém informers de Claims, XRs, MRs, Compositions e Events e enfileira o recurso afetado; um resync completo periódico cobre eventos perdidos.

## Consequências

- Baixa latência sem pressionar o API server.
- Exige leader election (única permissão de escrita: `leases` no namespace).
- Reinícios de watch são métricas/alertas (`crosswatc_informer_restarts_total`).

## Implementação

`internal/informer`, `internal/worker`
