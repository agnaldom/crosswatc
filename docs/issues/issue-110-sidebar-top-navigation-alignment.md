# Issue #110: Crossview-Accurate Sidebar & Top Navigation Alignment

## Status
- **State:** Open
- **Type:** Enhancement / Navigation
- **Target:** `apps/web/src/app/layout.tsx`, `apps/web/src/components/Sidebar.tsx`

## Context & User Story
Nas capturas de tela, a barra de navegação lateral (Sidebar) e a barra superior seguem a taxonomia exata do Crossview:
- **Cabeçalho:** Logo + botão de recolher `<`
- **Contexto:** Indicador de contexto (`CONTEXT: minikube` ou `in-cluster`)
- **Menu:**
  - `Dashboard`
  - `Providers`
  - `Compositions`
  - `XRDs`
  - `Composite Resources` (accordion expansível com Kinds dinâmicos como `XPostgres`, `XRedis`, `XNginx`, `XWebApp`)
  - `Claims`
  - `Managed Resources` (ou `Resources`)
  - `Settings` (com `Appearance` para alternar Dark/Light mode, etc.)
- **Rodapé:** "Logged in as **admin**" e link "Logout"
- **Top Bar:** "Crossplane Dashboard" e botão de busca com lupa no canto superior direito

**Como** usuário da plataforma,  
**Quero** navegar pela estrutura de recursos e menus exatamente como no layout padrão do Crossview,  
**Para que** a interface seja familiar, profissional e consistente com o ecossistema Crossplane.

## Acceptance Criteria
1. Seção de Contexto estilizada com o nome do contexto do cluster (`in-cluster` ou configurável).
2. Menu principal reorganizado com os links:
   - Dashboard
   - Providers (`/platform?tab=Providers`)
   - Compositions (`/platform?tab=Compositions`)
   - XRDs (`/platform?tab=XRDs`)
   - Composite Resources (accordion expansível que lista os Kinds retornados da API, destacando o Kind ativo)
   - Claims (`/infrastructure?view=claims`)
   - Managed Resources (`/infrastructure`)
   - Settings (com sub-opções ou modal de alternar tema)
3. Rodapé com informações do usuário autenticado ("Logged in as {user}") e ação de logout / alternância de perfil dev.
4. Top Header com título da página e botão de busca global no canto superior direito.
5. Suporte completo a tema Dark e Light mode com contraste de cores exato aos screenshots.
