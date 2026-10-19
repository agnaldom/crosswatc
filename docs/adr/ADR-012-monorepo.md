# ADR-012 — Monorepo

- **Status:** Aceito
- **Data:** 2026-10-17
- **Spec:** §9

## Contexto

UI, API, worker, contratos e deploy evoluem juntos; versões desalinhadas entre repositórios quebrariam o contrato de API.

## Decisão

Um repositório com `go.mod` único na raiz (`services/*` + `internal/*`), pnpm workspace para `apps/web`, migrations, Helm e docs. Um pipeline de CI valida Go, web, deploy (Helm, promtool, imagens) e carga.

## Consequências

- Mudanças de contrato atravessam backend e frontend no mesmo PR.
- CI único mais longo; jobs paralelos mitigam.
- Tipos TS espelham DTOs Go manualmente (geração a partir de OpenAPI é evolução futura).

## Implementação

raiz do repositório, `.github/workflows/ci.yml`
