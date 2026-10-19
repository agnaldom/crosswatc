# Issue #109: Right Slideout Drawer Inspector ("OnWatch Resources" & Status Conditions)

## Status
- **State:** Open
- **Type:** Feature / Drawer Component
- **Target:** `apps/web/src/components/OnWatchDrawer.tsx` & layout integration

## Context & User Story
Na captura de tela em modo claro do Crossview, há uma gaveta lateral retrátil (slideout drawer) no canto direito com um botão flutuante com ícone de olho (`Watching`). Ao clicar, o painel expande mostrando:
1. **OnWatch Resources:** Contador (`X`), botão de lixeira para limpar todos, e lista de recursos observados/fixados com pílula de status e botão `x` vermelho para desfixar.
2. **Status Conditions:** Lista de cartões com as condições ativas do recurso selecionado (`Responsive`, `Synced`, `Ready`), com status (`True` verde / `False` vermelho), Reason, Message detalhada do controller e Last Transition.
3. **Overview:** Resumo rápido com Name, UID, etc.

**Como** operador monitorando recursos críticos,  
**Quero** ter acesso rápido à lista de recursos sob observação (OnWatch) e aos detalhes de condições do recurso selecionado sem trocar de página,  
**Para que** eu possa acompanhar falhas de reconciliação e métricas de circuito em tempo real.

## Acceptance Criteria
1. Botão flutuante na lateral direita com ícone de olho e badge com quantidade de recursos no OnWatch.
2. Painel retrátil (drawer) animado que abre sobre a lateral direita.
3. Seção **OnWatch Resources**:
   - Cabeçalho com contagem e ícone de lixeira para limpar todos.
   - Lista de recursos observados: nome, kind, badge de status (`Not Synced`, etc.), e botão `x` para desfixar.
   - Clicar em um item da lista seleciona o recurso na tabela principal.
4. Seção **Status Conditions**:
   - Exibe as condições do recurso atualmente selecionado (`Responsive`, `Synced`, `Ready`, etc.).
   - Cartões com Type, Status pill (`True` / `False`), Reason, caixa de mensagem expandida para erros detalhados de reconciliação, e data/hora da última transição.
5. Seção **Overview**:
   - Exibe Name e UID do recurso selecionado.
6. Estado persistido no localStorage e integrado com o hook `useWatchlist()`.
