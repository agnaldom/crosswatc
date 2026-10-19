# ADR-009 — Autenticação via GitHub App

- **Status:** Aceito
- **Data:** 2026-10-17
- **Spec:** §44, §49

## Contexto

Tokens pessoais vinculam a automação a uma pessoa e têm escopo amplo demais.

## Decisão

API (escrita de PRs) e worker (leitura de PRs, checks, reviews e manifestos) usam um GitHub App instalado só no repositório de claims, com chave privada montada de Secret.

## Consequências

- Permissões mínimas (`contents`, `pull_requests`, `checks`, `statuses`, `metadata`).
- Rate limit por instalação; o worker cacheia manifestos para drift.
- Commits e PRs aparecem como o App; o solicitante fica no corpo do PR e na auditoria.

## Implementação

`internal/adapters/github`
