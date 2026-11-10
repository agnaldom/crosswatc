# Compatibilidade entre backend e web — 2026-10-21

A integração tem os contratos centrais alinhados após as correções desta revisão, mas ainda não representa todas as capacidades de um portal completo de infraestrutura. O backend atual rastreia recursos e suas projeções; não oferece um inventário independente de todos os XRs e managed resources do cluster.

## Escopo e evidência

Comparação das rotas registradas em `internal/httpapi/router.go`, DTOs, handlers de catálogo, formulários, mudanças, desativação, históricos, usuários e contextos com o cliente `apps/web/src/lib/api.ts`, tipos, telas e ponte SSE. Também foram lidos a autenticação, a consulta de recursos no store e a inicialização dos clientes Kubernetes.

Foi usado o grafo `crosswatc`, geração `2026-10-20T17:36:04Z`, com conferência de cobertura. O índice estava anterior às mudanças recentes e não incluía alguns arquivos novos; as conclusões foram verificadas diretamente no código atual. Esta é uma revisão dirigida dos contratos e fluxos web/API, não uma auditoria exaustiva de segurança, Kubernetes ou todas as regras de negócio.

## Incompatibilidades corrigidas

| Problema confirmado | Comportamento após correção |
| --- | --- |
| `view=claims/composites/managed` era enviado pelo web e ignorado pela API; os mesmos registros recebiam rótulos de entidades diferentes | Lista única de **Tracked resources**, com os campos reais do DTO. Claims, XRs e MRs podem ser inspecionados na topologia de cada recurso |
| `responsive` e `composition` eram filtros sem implementação no handler/store | Controles e parâmetros removidos do cliente. Permanecem os filtros implementados: tipo, prefixo de nome e condições Synced/Ready; estes últimos incluem `unknown` |
| Colunas `parentResource`, `deletionPolicy`, `resourceRefsCount` e `responsive` não existem no DTO da listagem | Removidas as suposições da tabela e do tipo. Política de remoção continua disponível nos nós reais da topologia |
| Inspector e aba Status inventavam `Responsive=True`, motivos de reconciliação e mensagens; watchlist podia usar `synced=true/ready=true` antes da leitura | Condições vêm da topologia, com loading/erro/ausência explícitos. Sem dados, nenhum estado positivo é fabricado |
| Pacotes recém-criados podem retornar `conditions:null`; a tela chamava `.find` sem proteção e tratava ausência como falha | Tipo aceita `null`; leitura compartilhada distingue True, False e Unknown, inclusive nos filtros e badges |
| Listagem limitada a 50 recursos e históricos consumiam somente a primeira página | Recursos, revisões, eventos e auditoria usam `nextCursor` e oferecem **Load more**, preservando o cursor e os registros já carregados |
| Ponte SSE esperava apenas `Page`, incompatível com histórico paginado | Atualização de eventos aceita cache paginado e preserva páginas anteriores; teste cobre deduplicação |
| Formulário dizia `editable=true` para SRE sem pertencer à squad, embora a mutação respondesse 403 | Backend informa read-only nesse perfil. Web usa a squad proprietária para habilitar editar, desativar e cancelar revisão |
| Alterações globais de kubeconfig eram permitidas a qualquer identidade autenticada | Switch/import/remove exigem a mesma autorização administrativa dos usuários; web esconde esses controles de membros comuns |
| Busca global exibia “nenhum resultado” quando a API falhava | Exibe o erro retornado pela API |
| Link de Composition enviava `apiVersion/kind` como filtro `kind` | Link extrai o Kind. Links do catálogo apontam para recursos rastreados, sem prometer inventário de MRs |
| A tela de usuários sugeria controle efetivo de login e acesso via registros locais | Texto esclarece que autenticação e squads vêm do provedor de identidade; consulta de usuários só é disparada para o perfil autorizado |

## Matriz dos fluxos

| Fluxo web | Contrato backend | Resultado da revisão |
| --- | --- | --- |
| Dashboard | `GET /dashboard` | DTO compatível; resumo limitado pelo backend conforme suas consultas |
| Recursos e tipos | `GET /resources`, `/resources/kinds` | Compatível como lista de recursos rastreados, com paginação |
| Estado, health e topologia | `GET /resources/:id/{state,health,topology}` | Compatível; condições do recurso vêm dos nós da topologia |
| YAML e eventos Kubernetes | `GET /resources/:id/{manifest,kube-events}` | Contratos e parâmetros compatíveis; dependem de acesso ao cluster e existência dos objetos |
| Histórico e auditoria | `GET /resources/:id/{revisions,events,audit}` e `/revisions/:rev` | Compatível com paginação; detalhe de revisão tem limites próprios no backend |
| Edição | `GET /form`, `POST /changes/preview`, `POST /changes` | Campos e payloads correspondem; autorização exige squad proprietária; submissão depende de GitOps |
| Cancelamento | `POST /revisions/:rev/cancel` | Cliente trata 204 sem tentar JSON; apenas revisões anteriores ao merge são canceláveis |
| Desativação | `POST /decommission/preview`, `POST /decommission` | Nomes e payloads correspondem; preservadas confirmação do nome, justificativa e regras do backend |
| Catálogo | `GET /platform/{providers,functions,configurations,xrds,compositions,mrds,mraps}` | Compatível, restrito a SRE. APIs Crossplane v2 ausentes usam `available=false` |
| Usuários | `GET/POST /users`, `PUT/DELETE /users/:id` | CRUD compatível com o registro local; não provisiona acesso no IdP |
| Contextos | `GET /contexts`, `/contexts/connection`; mutações de contextos | Payloads correspondem; alterações exigem admin/SRE. Existem limitações operacionais abaixo |
| Tempo real | `GET /stream` | Proxy preserva SSE e retomada; histórico paginado recebe eventos sem perder cursores |

## Limitações que permanecem

1. **Troca de cluster não é dinâmica.** `DefaultContextManager.SetContext` salva o kubeconfig; API e worker criam seus clientes uma vez no startup (`services/api/main.go`, `services/worker/main.go`). É necessário reiniciar ambos. Os registros no PostgreSQL não são separados por cluster. A tela e a sidebar agora distinguem contexto configurado de conexão efetiva. No Compose, o kubeconfig está montado como read-only; importação, remoção e troca precisam de uma configuração de armazenamento gravável antes de funcionarem nesse ambiente. Esta revisão não alterou esse volume nem trocou o cluster.
2. **Login/roles locais não controlam a autenticação atual.** O CRUD grava usuários e hashes, mas os authenticators são Dev/OIDC. Permissões de recursos derivam das squads do principal, e a administração usa SRE ou o identificador `admin`. Alterar `users.role` não altera claims/grupos do IdP. Não foi criado um novo mecanismo de login.
3. **Não há criação inicial de infraestrutura pelo web.** O backend expõe catálogo e mudanças sobre um recurso já rastreado; não registra uma rota para criar um recurso inicial pelo portal. Importação/descoberta do worker e criação de PRs de atualização são fluxos diferentes.
4. **GitOps e desativação dependem da configuração.** No Compose atual não há configuração de GitHub App nem ambientes habilitados para decommission. O backend pode responder 503 em submissões de PR e explicar o bloqueio da desativação. Contrato correto não significa que a funcionalidade esteja habilitada no ambiente.
5. **Formulários não cobrem todo OpenAPI.** O backend pode gerar arrays, objetos e chaves aninhadas; o renderer/validador web está orientado a escalares. Não foi implementado editor completo de schemas nesta revisão. Enums numéricos e chaves com pontos também merecem casos de integração específicos antes de declarar suporte geral a qualquer XRD.
6. **Cobertura funcional parcial de endpoints globais.** Backend tem operações e auditoria globais e catálogo de tipos/schemas/formulários. O web prioriza o histórico por recurso; não oferece telas dedicadas para todos esses endpoints. O detalhe de revisão do backend limita operações/eventos embutidos a 200; o novo Load more não muda esses limites internos.

## Validação

- `pnpm --filter web typecheck`
- `pnpm --filter web lint`
- `pnpm --filter web test`: 52 testes, incluindo regressões novas de contrato, Unknown e SSE paginado.
- `go test ./...`: suíte Go e regressões novas de autorização/formulário.
- Integração PostgreSQL executada em banco temporário isolado, sem modificar os recursos rastreados existentes.
- Imagens Docker da API e do web construídas e iniciadas com sucesso; container web saudável. Lista de recursos, inspector sem condições fabricadas e detalhe conferidos no Chrome após a atualização.
- Smoke test por `http://localhost:3000/api/v1`: `/me`, `/dashboard`, recursos (inclusive `ready=unknown`), tipos, usuários, contextos, estado, health, topologia, eventos, revisões e auditoria responderam 200. `limit=1` retornou `nextCursor` em recursos e eventos.
- Na primeira passagem, consultas de formulário e catálogo excederam 55 segundos; manifesto retornou 502, eventos Kubernetes retornaram 200 com lista vazia. A consulta de providers por `kubectl --context kind-crossplane-poc --request-timeout=10s` também encontrou timeout na descoberta de grupos, embora `/readyz` respondesse `ok`. A disponibilidade completa das leituras Kubernetes não foi comprovada; isso permanece como investigação operacional separada dos contratos e testes que passaram.

Não foram criados usuários, trocados contextos, canceladas revisões nem abertos PRs reais para validar a interface. Esses caminhos foram exercitados com os fakes e testes do backend; o teste no ambiente em execução se limita a leituras.
