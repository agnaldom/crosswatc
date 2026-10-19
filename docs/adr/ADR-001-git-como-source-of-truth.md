# ADR-001 — Git como source of truth

- **Status:** Aceito
- **Data:** 2026-10-17
- **Spec:** §3.1, §41–44

## Contexto

O Crosswatc precisa permitir mudanças de infraestrutura sem criar um segundo caminho de escrita que fuja da revisão, do CI e do histórico do repositório GitOps.

## Decisão

Toda mudança vira commit + Pull Request no repositório de claims via GitHub App. O portal nunca escreve em Cloud, Claims, XRs, MRs ou Applications do Argo CD.

## Consequências

- Auditoria, aprovação e rollback ficam no Git, que a organização já usa.
- Latência maior: a mudança só vale após merge → sync do Argo → reconciliação do Crossplane.
- Divergência Git × cluster bloqueia novas mudanças (ADR-003, `internal/httpapi/consistency.go`).

## Implementação

`internal/gitops`, `POST /resources/{id}/changes`
