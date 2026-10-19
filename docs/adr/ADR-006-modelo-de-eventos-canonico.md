# ADR-006 — Modelo de eventos canônico

- **Status:** Aceito
- **Data:** 2026-10-17
- **Spec:** §33–35

## Contexto

GitHub, Argo CD, Crossplane e Backstage descrevem o mesmo ciclo com vocabulários diferentes; a UI e a auditoria precisam de uma linha do tempo única.

## Decisão

Normalizar tudo em eventos `DOMÍNIO.ENTIDADE.AÇÃO` (ex.: `GITOPS.PR.MERGED`, `INFRA.RECONCILE.READY`) com correlation ID, gravados append-only com `dedupe_key`. Webhooks entram por uma `inbox` idempotente e são gatilho, não verdade: o worker relê a fonte.

## Consequências

- Timeline e métricas independem da ferramenta de origem.
- Reentregas e resyncs não duplicam eventos.
- Novos eventos exigem atualizar `internal/events`.

## Implementação

`internal/events`, `internal/worker/inbox.go`
