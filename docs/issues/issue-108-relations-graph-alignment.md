# Issue #108: Interactive Relations DAG Graph Alignment (`@xyflow/react`)

## Status
- **State:** Open
- **Type:** Feature / Visual Refinement
- **Target:** `apps/web/src/components/TopologyView.tsx`, `apps/web/src/components/topology/*`

## Context & User Story
Nas capturas de tela do Crossview, a aba **Relations** exibe uma visualização em grafo dos recursos relacionados ao recurso selecionado, com layout hierárquico bem definido:
- Nós superiores: Composite Resource (Owner), Claim (Owner)
- Nó central em destaque: O recurso atual (com borda/glow azul e badge de status)
- Nós laterais e inferiores: Managed Resources gerados e Composição utilizada
- Linhas tracejadas dinâmicas (dashed blue lines) conectando os nós

**Como** usuário inspecionando uma infraestrutura Crossplane,  
**Quero** ver o grafo de relações organizado exatamente como na referência visual do Crossview,  
**Para que** eu identifique instantaneamente os pais (Claim/XR Owner), filhos (Managed Resources) e a composição associada.

## Acceptance Criteria
1. O nó central selecionado deve ter destaque visual (borda azul, sombra/glow sutil).
2. As conexões entre os nós devem usar linhas tracejadas (`stroke-dasharray: 4 4`), na cor do tema (azul elétrico).
3. Cada nó exibe: Tipo/Papel no topo (`Composite Resource (Owner)`, `Claim (Owner)`, `Managed Resource`, `Composition`), nome do recurso, namespace quando aplicável e indicador de saúde.
4. Clicar em nós navegáveis permite inspecionar detalhes do nó correspondente.
5. Suporte a tema claro e escuro consistente com o background do canvas.
