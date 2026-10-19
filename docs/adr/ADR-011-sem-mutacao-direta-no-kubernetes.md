# ADR-011 — Sem mutação direta no Kubernetes

- **Status:** Aceito
- **Data:** 2026-10-17
- **Spec:** §3.1, §27

## Contexto

Qualquer permissão de escrita em infraestrutura abriria um caminho que contorna Git, aprovação e auditoria.

## Decisão

A ServiceAccount do Crosswatc tem apenas `get/list/watch` em recursos de infraestrutura. A única escrita é `leases` no próprio namespace para leader election. Secrets são lidos só por metadata.

## Consequências

- Comprometimento da API/worker não altera infraestrutura.
- Correções emergenciais continuam fora do portal (por design).
- `secrets: get` é necessário para checar existência; documentado como limitação de RBAC.

## Implementação

`deploy/helm/crosswatc/templates/rbac.yaml`
