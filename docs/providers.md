# Providers: configurações ativas e catálogo

A página `/providers` apresenta duas visões:

- **Active configurations**: metadados de ProviderConfig/ClusterProviderConfig de providers com Installed=True e Healthy=True. As APIs são descobertas pelos CRDs estabelecidos pertencentes à revisão atual do provider. Configurações de revisões antigas e objetos em exclusão não aparecem. Nenhum Secret ou valor de credencial é retornado.
- **Provider catalog**: Azure, AWS, GCP, Kubernetes, Helm e SQL, com documentação, versão explícita e estado real da instalação. As versões sugeridas foram consultadas nos releases oficiais em 21/10/2026; confirme a compatibilidade com a versão do Crossplane antes de instalar.

Azure, AWS e GCP são apresentados como bundles completos por nuvem, em vez de cards separados por serviço. Cada bundle inclui o `provider-family-*` e todos os pacotes de serviços listados no diretório `cmd/provider` do release oficial, mapeando `config` para `provider-family-*` e excluindo os comandos raiz e monolith. A lista está versionada em `internal/httpapi/provider_bundles.go`:

| Provider | Release | Pacotes |
| --- | --- | --- |
| Azure | v2.8.0 | 92 |
| AWS | v2.9.0 | 181 |
| GCP | v3.1.0 | 82 |

As versões desses bundles são fixas para manter coerência com o inventário de pacotes; Kubernetes, Helm e SQL permitem editar a versão. O diálogo mostra a lista completa e o consumo potencial de CPU/memória antes de instalar. Nenhum provider cloud é instalado automaticamente ao abrir o catálogo. Referências: [Crossplane Providers](https://docs.crossplane.io/latest/packages/providers/), [Azure release](https://github.com/crossplane-contrib/provider-upjet-azure/tree/v2.8.0/cmd/provider), [AWS release](https://github.com/crossplane-contrib/provider-upjet-aws/tree/v2.9.0/cmd/provider), [GCP release](https://github.com/crossplane-contrib/provider-upjet-gcp/tree/v3.1.0/cmd/provider).

## API

- GET `/api/v1/platform/provider-configs`
- GET `/api/v1/platform/provider-catalog`
- POST `/api/v1/platform/provider-catalog/:provider/install`, corpo `{"version":"v2.8.0"}` para Azure, ou o release correspondente do catálogo

O catálogo exige o perfil SRE/plataforma. Instalar exige também a role administrativa explícita. O backend aceita somente os repositórios presentes no catálogo e versões explícitas, cria os objetos Provider do bundle e responde 202 com created/existing/total. O diálogo consulta o progresso real durante a criação dos pacotes. A reconciliação dos controllers pelo Crossplane é assíncrona; o estado do card vem das condições reais. Objetos existentes não são atualizados ou substituídos. Em falhas parciais, os pacotes já criados permanecem e uma nova tentativa instala somente os que faltam. O processamento respeita o cancelamento da conexão e tem limite de dois minutos por tentativa. Antes de criar qualquer objeto de um bundle cloud, a API verifica os providers existentes. Um mesmo pacote com outra versão ou publisher retorna 409 com o conflito, evitando misturar silenciosamente gerações diferentes de providers.

Essa é uma exceção limitada à leitura de infraestrutura: o chart inclui uma ClusterRole separada com **create apenas em pkg.crossplane.io/providers**. Não concede update, patch ou delete e não grava ProviderConfigs, Secrets ou recursos cloud. A API registra o ator no log e na anotação do Provider. Quando o chart cria as contas, API e worker usam service accounts separados: o worker conserva acesso somente de leitura à infraestrutura e a Lease de eleição. Instalar controllers não configura acesso à nuvem; credenciais e ProviderConfigs continuam sob gestão do operador/GitOps.

Em instalações com RBAC externo, conceda essa permissão explicitamente ao service account da API. Se não houver permissão, a instalação retorna 403 e o web mostra o erro. Não é necessário aplicar nenhuma instalação para usar o catálogo como consulta.

## Navegação

Cada item do menu de plataforma possui sua própria página: `/providers`, `/functions`, `/configurations`, `/compositions`, `/resource-definitions`, `/managed-resource-definitions` e `/activation-policies`. O título, o breadcrumb e o item ativo do menu acompanham a rota atual. Não há barra de abas compartilhada entre essas páginas. Links antigos `/platform?tab=...` respondem com redirecionamento HTTP 307 para a página correspondente; `/platform` abre `/providers`.

## Acompanhar instalação e logs

No card do catálogo, **View installation & logs** abre `/providers/<id>/installation`, por exemplo `/providers/azure/installation`. O link permanece disponível depois que os objetos já foram adicionados. A página separa pacotes prontos, aguardando instalação/dependências, com erro e ainda não adicionados; consulta o estado real a cada cinco segundos. Todos os objetos de providers correspondentes aparecem, incluindo versões/publishers duplicados, com um aviso de conflito.

Ao selecionar um pacote, a página mostra condições e mensagens do Provider e de sua revisão atual, eventos Kubernetes associados por UID, pods do runtime, estado/reinícios dos containers e saúde do controlador Crossplane. Leituras opcionais com erro são exibidas como avisos, sem apagar as condições que puderam ser lidas. O último snapshot de pacotes é preservado quando uma atualização falha.

- GET `/api/v1/platform/providers/:name/diagnostics`: SRE/plataforma.
- GET `/api/v1/platform/providers/:name/logs`: exige também administrador.

Logs são lidos somente em `crossplane-system`, com as permissões Kubernetes já existentes; esta funcionalidade não amplia RBAC. O navegador não escolhe namespaces, pods ou containers arbitrários. O backend deriva os pods do provider/revisão e do controlador Crossplane, limita cada leitura a 200 linhas e 64 KiB, e filtra o log compartilhado para o pacote selecionado e falhas gerais do controlador (como perda de eleição de líder). Logs anteriores do Crossplane são consultados quando há reinícios. Sem acesso a `pods/log`, o erro de permissão aparece na página. Instalações do Crossplane em outro namespace precisam de configuração explícita antes de habilitar essa leitura.

Diagnóstico local em 21/10/2026: coexistem a família Azure v1.3.0 e pacotes Azure v2.8.0 aguardando dependências. Os logs anteriores do Crossplane registraram timeouts da API Kubernetes e `cannot start controller manager: leader election lost`. A tela mostra esses bloqueios; adicionar objetos Provider não indica que os controllers ficaram prontos. A instalação existente exige revisão de compatibilidade e da estabilidade do cluster antes de ser considerada concluída.

Para acompanhar a submissão dos pacotes pelo terminal, use `docker compose logs -f api`. A API registra o início, cada pacote adicionado, o total, falhas e conflitos de versão, com ator e correlation ID. O estágio `awaiting_crossplane` significa que os objetos foram criados e ainda dependem da reconciliação do Crossplane; os eventos e logs do controlador na página mostram essa etapa posterior.
