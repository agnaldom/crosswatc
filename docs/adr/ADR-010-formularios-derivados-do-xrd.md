# ADR-010 — Formulários derivados do XRD

- **Status:** Aceito
- **Data:** 2026-10-17
- **Spec:** §38

## Contexto

Manter formulários à mão para cada tipo de recurso não escala e diverge do schema real.

## Decisão

O formulário vem do `openAPIV3Schema` de `spec.parameters` do XRD, enriquecido por metadata UX (annotation `crosswatc.io/ui-metadata` ou extensão `x-crosswatc`). Campos são somente leitura por padrão; imutáveis (`self == oldSelf`) nunca são editáveis.

## Consequências

- Plataforma controla o que é editável no próprio XRD.
- Validação idêntica no cliente (Zod) e no servidor (`internal/changes`).
- Schemas sem metadata ficam visíveis porém não editáveis.

## Implementação

`internal/forms`, `apps/web/src/lib/form-schema.ts`
