# ADR-003 — Resource State separado de Revision State

- **Status:** Aceito
- **Data:** 2026-10-17
- **Spec:** §12, §14, §32

## Contexto

Um recurso pode estar saudável enquanto uma mudança aguarda aprovação; um único status confunde "como está" com "o que está mudando".

## Decisão

`ResourceState` (derivado de Claim/XR/MR) e `RevisionState` (jornada de uma mudança) são modelados e persistidos separadamente; no máximo uma revision não terminal por recurso (índice único parcial).

## Consequências

- UI mostra `Operational` + `Revision 8: WAITING_APPROVAL` sem ambiguidade.
- Regras de derivação vivem em código puro e testado por tabela (`internal/lifecycle`).
- Transições de revision são monotônicas; cancelamento/substituição são explícitos.

## Implementação

`internal/lifecycle`, `db/migrations/00001_init.sql`
