# ADR-002 — Arquitetura single-cluster

- **Status:** Aceito
- **Data:** 2026-10-17
- **Spec:** §3.4, §26

## Contexto

Crossplane e Argo CD rodam num único cluster de controle. Suporte multi-cluster (seletor de contexto, registry, cache por cluster) multiplicaria a complexidade sem demanda.

## Decisão

O backend usa `rest.InClusterConfig()` em produção e `KUBECONFIG` (contexto atual) em desenvolvimento. Não há troca de contexto nem registro de clusters.

## Consequências

- Um único conjunto de informers, um RBAC, um cache.
- Expandir para multi-cluster exigirá rever discovery, informers e chaves (`claimKey`).

## Implementação

`internal/kube/client.go`
