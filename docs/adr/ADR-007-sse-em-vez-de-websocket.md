# ADR-007 — SSE em vez de WebSocket

- **Status:** Aceito
- **Data:** 2026-10-17
- **Spec:** §45, §66

## Contexto

O fluxo em tempo real é servidor → navegador; ações do usuário já são REST.

## Decisão

`GET /api/v1/stream` com Server-Sent Events, filtrado pela mesma AuthZ da API, `Last-Event-ID` para retomada e fan-out entre réplicas via `LISTEN/NOTIFY` do PostgreSQL. A UI cai para polling de 15 s se o stream falhar.

## Consequências

- Funciona sobre HTTP comum (proxies, auth, CSP `connect-src 'self'`).
- Sem canal cliente → servidor; não necessário.
- Proxies precisam de `X-Accel-Buffering: no`.

## Implementação

`internal/stream`, `internal/httpapi/stream.go`
