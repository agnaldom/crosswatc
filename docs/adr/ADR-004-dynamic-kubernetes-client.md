# ADR-004 — Dynamic Kubernetes client

- **Status:** Aceito
- **Data:** 2026-10-17
- **Spec:** §21–23

## Contexto

Claims, XRs e Managed Resources pertencem a grupos que mudam com cada XRD e provider; clients tipados exigiriam recompilar a cada novo tipo.

## Decisão

Usar `dynamic.Interface` + discovery de XRDs (claim/XR) e MRDs (com fallback para CRDs na categoria `managed`). Nada é hardcoded por provider.

## Consequências

- Novos tipos aparecem sem deploy.
- Validação de forma fica a cargo do parsing de `unstructured`; erros viram `Errors`/`Missing` na discovery, sem derrubá-la.
- RBAC precisa listar os grupos (não há wildcard por sufixo).

## Implementação

`internal/kube`
