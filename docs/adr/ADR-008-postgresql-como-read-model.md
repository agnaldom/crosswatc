# ADR-008 — PostgreSQL como read model

- **Status:** Aceito
- **Data:** 2026-10-17
- **Spec:** §37

## Contexto

A UI precisa responder em < 400 ms (p95) com histórico, filtros e paginação, sem consultar Kubernetes/GitHub por request.

## Decisão

O worker projeta estado, topologia, revisions, operations e eventos no PostgreSQL; a API lê só do banco (exceto catálogo de XRDs). Optimistic locking (`version`) no reconciler; grants tornam `tracker_event` e `audit_log` append-only.

## Consequências

- Leituras rápidas e independentes da disponibilidade das fontes (k6: p95 < 1 ms em 5000 recursos).
- Consistência eventual: o read model atrasa o mundo real pelo lag de ingestão/reconciliação.
- Migrations versionadas (goose) com dono único: o worker.

## Implementação

`internal/store`, `db/migrations`
