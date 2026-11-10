# Revisão do backend crossview — Keycloak/OIDC

Data: 2026-10-21. Projeto analisado: `/Users/agnaldo.marinho/projeto/crossview`.

O projeto tem login OIDC integrado à sessão local, mas precisa corrigir os pontos de autenticação e autorização abaixo antes de servir de referência para uso em produção. Esta revisão não alterou o código do crossview, configurações do Keycloak, contas ou sessões reais.

## Fluxo encontrado

1. O web consulta `/api/auth/sso/status` e oferece um link para `/api/auth/oidc`.
2. O backend faz discovery e redireciona para o Keycloak usando Authorization Code e client confidencial.
3. No callback, troca o código por access token e consulta UserInfo.
4. Procura/cria um usuário no banco e grava `userId` e `userRole` em cookie de sessão.
5. As próximas chamadas usam o cookie. `RequireAdmin` consulta `users.role`; não consulta roles do Keycloak.

Referências: `services/sso_service.go`, `api/controllers/sso/sso_controller.go`, `api/middlewares/session_auth.go`, `src/domain/services/AuthService.js`, dentro do projeto crossview.

## Achados prioritários

### 1. Alta — State gerado, mas não validado

`crossview-go-server/services/sso_service.go:89` gera e envia state. O callback recebe o parâmetro em `:105`, mas não o compara com uma transação de login armazenada. O controller também apenas o repassa (`api/controllers/sso/sso_controller.go:77`).

Isso deixa o retorno sem a vinculação esperada ao navegador que iniciou o login, permitindo cenários de login CSRF. A correção precisa armazenar state por transação, com validade curta e consumo único, e rejeitar ausente, incorreto, expirado e reutilizado. A implementação também não envia PKCE nem nonce; adotar Code + PKCE S256 e validação do nonce no ID Token.

### 2. Alta — Resposta OIDC usada sem validação do ID Token

`services/sso_service.go:182` extrai apenas `access_token`. Não valida um `id_token`, seu issuer, audience e expiração, nem compara o subject do UserInfo com o do ID Token. O teste de sucesso em `services/sso_service_test.go:238` simula uma resposta com apenas access token, e em `:280` usa state arbitrário sem login iniciado previamente.

O fato de o servidor consultar UserInfo não significa que um JWT arbitrário seja aceito diretamente: o endpoint do provedor ainda valida o access token. O problema confirmado é a ausência das validações do cliente OIDC. Usar uma biblioteca OIDC e conferir o vínculo entre a autenticação e o perfil retornado.

### 3. Alta — Vinculação automática de conta por e-mail/username

`models/user.go:192` procura primeiro por e-mail, depois por username (`:199`). O `sub` obtido em `services/sso_service.go:215` aparece somente no log; não é armazenado como identidade externa.

Uma identidade SSO com atributos coincidentes pode ser vinculada a uma conta local existente e herdar sua role, inclusive administrativa. O impacto depende de quem pode criar/alterar esses atributos no realm e de possíveis colisões; não foi realizada tentativa contra o Keycloak real. Persistir a chave única `(issuer, subject)` e exigir um processo explícito e autorizado para vincular contas existentes. Não confiar somente em e-mail, mesmo que verificado, como identificador imutável.

### 4. Alta quando defaults são utilizados — Sessão e login local mantêm caminhos independentes do Keycloak

`lib/env.go:117` fornece uma chave de sessão padrão conhecida e defaults de credenciais administrativas. `commands/serve.go:63` cria admin local quando necessário; `api/controllers/auth/auth_controller.go:131` autentica a senha local sem condicionar isso ao uso exclusivo do SSO. `api/middlewares/session.go:52` usa a chave configurada para assinar cookies.

Se os defaults forem mantidos, há risco de falsificação de sessão e acesso administrativo por senha local, contornando as exigências de MFA e políticas do IdP. Isso é uma conclusão condicional sobre a configuração; não foi inspecionado o secret real de um deployment. Em produção, rejeitar defaults/segredos ausentes e tornar login local um modo explícito ou um mecanismo de recuperação restrito.

### 5. Alta — Alterações do cluster sem autorização administrativa

`api/routes/kubernetes.go:38` e `:46` protegem troca/importação/remoção de contextos com `AuthMiddleware`, que no modo sessão verifica apenas a existência de `userId` (`api/middlewares/session_auth.go:31`). Não há `RequireAdmin` nessas rotas.

No escopo dessas rotas, um usuário autenticado comum pode solicitar mudanças no kubeconfig compartilhado. O backend precisa autorizar essas mutações explicitamente; ocultar controles no web não resolve a autorização.

### 6. Alta/condicional — CORS permite qualquer origem com credenciais

`api/middlewares/cors.go:29` habilita credenciais e aceita todas as origens. Isso amplia a exposição de respostas autenticadas a origens não confiáveis quando os cookies forem enviados. O comportamento concreto depende de SameSite, domínio e política do navegador; não foi demonstrada exploração em deployment real.

Usar allowlist de origens, proteção CSRF nas mutações com cookie e uma política SameSite explícita adequada à topologia do login. A lista atual de métodos CORS também não contém DELETE, embora existam rotas DELETE de contextos; isso pode quebrar clientes web em origem distinta.

### 7. Média — Permissões e logout não acompanham o IdP

O papel local é determinado pelo banco (`models/user.go:210`, `api/middlewares/session_auth.go:73`); o fluxo não lê `realm_access.roles`, `resource_access` ou grupos do Keycloak. Se o banco estiver vazio, `FindOrCreateSSOUser` promove o primeiro usuário a admin; normalmente o bootstrap já cria um admin, portanto essa condição depende do estado inicial.

A sessão usa MaxAge de 24 horas (`api/middlewares/session.go:56`). O middleware aceita seu `userId` sem revalidar a sessão do IdP ou a existência do usuário no banco para as rotas comuns. Desabilitar uma conta no Keycloak não invalida automaticamente esse cookie. O logout em `api/controllers/auth/auth_controller.go:174` limpa somente a sessão local.

Definir uma fonte de autorização, sincronizar ou avaliar roles explicitamente, implementar logout OIDC e, conforme o requisito, invalidação de sessão por backchannel/revogação. A role administrativa local já é consultada no banco por `RequireAdmin`, mas isso não integra a revogação do Keycloak.

### 8. Média — Callback instável e rede sem limites

`api/controllers/sso/sso_controller.go:151` deriva o callback de Host/X-Forwarded-Host e X-Forwarded-Proto. Em requisições normais, a URL OIDC configurada não é a fonte principal. Proxies ou cabeçalhos indevidos podem quebrar a correspondência com redirect URIs; o import do realm limita o callback a localhost. Usar URL pública canônica e confiar em cabeçalhos somente de proxies controlados.

`services/sso_service.go:63`, `:171` e `:195` usam o cliente HTTP padrão, sem timeout próprio e sem vincular as requisições ao context recebido. O discovery é repetido. Um IdP lento pode manter requisições pendentes. Usar cliente com timeout, context e discovery gerenciado/cacheado.

## Configuração fornecida no repositório

- Compose usa Keycloak `latest`, `start-dev` e HTTP. É configuração de desenvolvimento, não base de produção.
- O client OIDC importado é confidencial, com Standard Flow ligado e Implicit Flow desligado. Direct Access Grants também está ligado, embora o fluxo analisado não o utilize.
- O Compose monta `config/examples/config-session.yaml.example`, onde OIDC está desabilitado. Subir apenas o serviço Keycloak não habilita o login OIDC no aplicativo.
- O client/realm importado não define mappers próprios de grupos/roles; defaults do Keycloak e configurações posteriores precisam ser verificados no ambiente real.
- Existe opção SAML no web, mas `services/sso_service.go:245` retorna explicitamente que o callback não foi implementado. Não considerar essa integração funcional.

## Comparação e direção para crosswatc

| Aspecto | crossview | crosswatc atual |
| --- | --- | --- |
| Login web | OIDC Code seguido de cookie local | Proxy de API; não inicia login OIDC por conta própria |
| Validação de tokens | Fluxo manual/UserInfo, sem ID Token | `internal/auth/auth.go:111` usa discovery e verifier com audience |
| Autorização | Role do usuário no banco | Squads e grupo SRE derivados das claims |
| Keycloak roles | Não consumidas pelo fluxo | Leitura de claim de primeiro nível; não percorre automaticamente roles aninhadas |
| Ambiente Compose | Keycloak de desenvolvimento, OIDC da app desligado no perfil montado | Identidade Dev; não exercita autenticação Keycloak |

Aproveitar o padrão de manter os tokens fora do JavaScript e a sessão no backend, com implementação OIDC validada. Para crosswatc:

1. Preservar a validação JWT de issuer/audience/expiração. Acrescentar login web por um BFF ou proxy OIDC corretamente configurado, mantendo a API protegida e sem headers Dev em produção.
2. Definir audience da API e um mapper no Keycloak que emita o claim esperado. Exemplo de contrato possível: `groups: ["squad-payments", "platform-sre"]`. Se usar full group paths, normalizar caminhos deliberadamente; o parser atual espera prefixos como `squad-`.
3. Se optar por client roles, implementar leitura explícita de `resource_access.<client>.roles`; configurar `CROSSWATC_OIDC_GROUPS_CLAIM=realm_access.roles` não resolve sozinho, pois hoje o código faz acesso por chave direta.
4. Remover a autorização por `p.User == "admin"` em `crosswatc/internal/httpapi/router.go:199`. Um nome de usuário coincidente não deve conceder administração no modo OIDC. Usar permissão/role administrativa explícita, separada de identidade e do simples acesso global de leitura.
5. Definir issuer público canônico acessível ao navegador e aos serviços, clients/audiences, redirect URI exata, logout e duração/revogação de sessões. Não desabilitar validação de issuer/audience para contornar diferenças entre localhost e nomes internos do Docker.

## Validação e limites

Revisão dirigida dos caminhos de SSO, sessão, usuários e rotas de Kubernetes, com comparação da autenticação do crosswatc. Grafo crossview na geração `2026-10-20T23:13:01Z`, cobertura conferida e fontes lidas diretamente. Não é uma auditoria exaustiva de toda a aplicação ou da instalação efetiva de Keycloak.

Os testes existentes passaram, executados com `-mod=readonly`, SQLite em memória e endpoints HTTP simulados:

- `go test -mod=readonly ./services -run '^TestSSOService_'`
- `go test -mod=readonly ./api/controllers/sso ./api/middlewares -run 'TestSSOController_|TestSessionAuthMiddleware_'`

O teste de sucesso existente aceita a resposta simulada sem ID Token e um state sem transação iniciada previamente. Passar na suíte atual não comprova as garantias de segurança ausentes no código. Não foi feito login real nem alteração de realm, roles, credenciais ou usuários. Arquivos de runtime com segredos não foram usados para presumir a configuração real.

## Referências oficiais

- [OpenID Connect Core — validação do ID Token](https://openid.net/specs/openid-connect-core-1_0.html#IDTokenValidation).
- [OpenID Connect Core — identidade estável por issuer e subject](https://openid.net/specs/openid-connect-core-1_0.html#ClaimStability).
- [Keycloak — endpoints OIDC e logout](https://www.keycloak.org/securing-apps/oidc-layers).
- [Keycloak — administração, roles e audience mappers](https://www.keycloak.org/docs/26.7.0/server_admin/index.html).
