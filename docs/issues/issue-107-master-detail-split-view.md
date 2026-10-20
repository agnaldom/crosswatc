# Issue #107: Split-Pane Master-Detail View on Resource Explorer (`/infrastructure`)

## Status
- **State:** Open
- **Type:** Feature / UX
- **Target:** `apps/web/src/app/infrastructure/page.tsx` & detail components

## Context & User Story
Conforme a referência visual do Crossview (screenshots dark/light mode), os usuários precisam explorar e gerenciar recursos com alta agilidade sem perder o contexto da listagem de recursos. Atualmente, clicar em um recurso navega para uma página separada (`/infrastructure/[id]`).

**Como** desenvolvedor ou operador de plataforma,  
**Quero** visualizar a lista de recursos e, ao selecionar uma linha, abrir o painel de detalhes diretamente abaixo da tabela (Master-Detail na mesma tela),  
**Para que** eu possa inspecionar Overview, YAML, Status, Relations e Events de forma rápida e contextual.

## Acceptance Criteria
1. **Filtros e Contagem:**
   - Barra de filtros com busca textual (`Search...`), filtro de status (`All Statuses`) e indicador de contagem (`X results` à direita).
2. **Tabela de Recursos:**
   - Colunas: `NAME`, `KIND`, `STATUS` (com pílulas combinadas: `Synced` em verde, `Ready`/`Not Ready` em vermelho, `Responsive`), `CREATED` (data formatada).
   - Linha selecionável: clicar em uma linha ativa o estado selecionado com destaque visual e sincroniza a URL (`?selected=<id>`).
3. **Painel de Detalhes Inline (Split View):**
   - Abre diretamente abaixo da tabela ao selecionar um recurso.
   - Cabeçalho: Kind e apiVersion (`XPostgres example.org/v1alpha1`), nome do recurso, botão `Watching`/`Watch` e botão de recolher `-`.
   - Abas: `Overview`, `YAML`, `Status`, `Relations`, `Events`.
   - A aba `Relations` exibe o grafo de topologia (`@xyflow/react`) inline.
   - A aba `YAML` exibe o manifesto completo formatado com syntax highlight e botão de copiar.
   - A aba `Status` exibe o detalhamento de condições.
   - A aba `Events` exibe os eventos Kubernetes/Lifecycle.
   - A aba `Overview` exibe resumo de metadados, owner, claim e composition.
