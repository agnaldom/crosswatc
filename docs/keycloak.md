# Login com Keycloak

O web usa Authorization Code + PKCE S256. A API valida state de uso único, vínculo ao navegador, nonce, assinatura, issuer, audience, expiração e correspondência de subject entre ID Token e access token. Tokens ficam no processo da API; o navegador recebe um identificador de sessão opaco em cookie HttpOnly, SameSite=Lax e Secure quando a URL pública usa HTTPS.

## Ambiente local completo

```sh
python3 deploy/keycloak/setup.py
docker compose -f docker-compose.yml -f docker-compose.keycloak.yml up -d --build
```

- Aplicação: http://localhost:3000
- Keycloak: http://keycloak.localhost:18080
- Realm: `crosswatc`
- Client web confidencial: `crosswatc-web`
- Audience da API: `crosswatc-api`
- Usuário local da aplicação: `crosswatc-dev`
- Senha desse usuário: campo `CROSSWATC_LOCAL_USER_PASSWORD` em `.env.keycloak`.
- Conta administrativa do sandbox Keycloak: `KC_BOOTSTRAP_ADMIN_USERNAME` e `KC_BOOTSTRAP_ADMIN_PASSWORD` no mesmo arquivo.

O script gera os segredos uma única vez, com permissão 0600, e não sobrescreve um arquivo existente. Não compartilhe nem versione `.env.keycloak`. O realm usa interpolação das variáveis do container para receber os valores. O usuário e as permissões amplas desse sandbox servem somente para desenvolvimento.

O hostname `keycloak.localhost` aponta para o loopback no navegador e é um alias DNS na rede Docker. O issuer permanece idêntico dentro e fora dos containers, sem ignorar sua validação. O overlay fixa a porta do web em 3000, que corresponde à redirect URI exata do realm. Keycloak usa H2 persistente e `start-dev` neste sandbox; não é a configuração de produção.

Para voltar à identidade local Dev:

```sh
docker compose up -d --no-deps --force-recreate api web
```

O modo Dev requer `CROSSWATC_AUTH_MODE=dev` tanto na API como no web. Headers Dev não são injetados no modo OIDC, mesmo que variáveis antigas continuem presentes. Produção rejeita o modo Dev.

## Configurar um Keycloak existente

Na API:

```ini
CROSSWATC_AUTH_MODE=oidc
CROSSWATC_OIDC_ISSUER=https://identity.example.com/realms/crosswatc
CROSSWATC_OIDC_AUDIENCE=crosswatc-api
CROSSWATC_OIDC_CLIENT_ID=crosswatc-web
CROSSWATC_OIDC_CLIENT_SECRET=<secret montado de forma segura>
CROSSWATC_PUBLIC_URL=https://workspace.example.com
CROSSWATC_SESSION_TTL=8h
CROSSWATC_ADMIN_GROUP=platform-admin
CROSSWATC_SRE_GROUP=platform-sre
CROSSWATC_SQUAD_GROUP_PREFIX=squad-
CROSSWATC_OIDC_GROUPS_CLAIM=groups
CROSSWATC_OIDC_ROLES_CLIENT_ID=crosswatc-api
```

No web, configure `CROSSWATC_AUTH_MODE=oidc` e `CROSSWATC_API_URL` com a origem interna da API. Não coloque o client secret ou tokens em variáveis `NEXT_PUBLIC_*`.

No client web do Keycloak:

- Client authentication ligado; Standard Flow ligado; PKCE S256 obrigatório.
- Implicit Flow e Direct Access Grants desligados.
- Redirect URI exata: `https://workspace.example.com/api/v1/auth/callback`.
- Post logout redirect URI exata: `https://workspace.example.com`.
- Audience mapper adicionando `crosswatc-api` ao **access token**; o ID Token mantém a audience do client web.
- Scopes `openid profile email` e mappers que emitam os grupos/roles usados pela aplicação.
- Backchannel logout URL: `https://workspace.example.com/api/v1/auth/backchannel-logout`, ou a URL interna equivalente da API se acessível ao Keycloak. Configure envio do session ID.

Issuer, callback e logout são canônicos e não são derivados de Host/X-Forwarded-Host enviados pelo cliente. HTTPS é exigido para issuer e URL pública na configuração de produção do login web. O proxy/ingress deve preservar os cookies, Location e Origin; as rotas de login devem chegar ao web ou à mesma instância da API.

## Permissões

A API aceita os grupos do claim configurado, realm roles em `realm_access.roles` e client roles somente de `resource_access.<CROSSWATC_OIDC_ROLES_CLIENT_ID>.roles`.

| Grupo/role | Permissão |
| --- | --- |
| `squad-payments` | Leitura e solicitações de mudança dos recursos pertencentes à squad `payments` |
| `platform-sre` | Leitura global e catálogo da plataforma; não concede escrita em recursos de outras squads |
| `platform-admin` | Administração dos registros locais de usuários e dos contextos Kubernetes |

Grupos com caminho `/squad-payments` são aceitos. Caminhos aninhados, como `/organization/squad-payments`, devem ser mapeados para nomes correspondentes ao contrato. Roles de clients diferentes do configurado são ignoradas. O username `admin` não concede privilégios. A identidade autenticada inclui issuer e subject; não há vinculação automática a registros locais por e-mail ou username. O rate limit utiliza a identidade estável issuer+subject.

A role no CRUD local de usuários não modifica grupos, senha ou acesso no Keycloak. Não há endpoint de login com senha local que contorne o IdP. Para desenvolver sem IdP, o modo Dev usa headers explícitos `X-Dev-User`, `X-Dev-Squads`, `X-Dev-SRE` e `X-Dev-Admin`.

## Sessão, CSRF e logout

- Mutações autenticadas por cookie exigem Origin igual à URL pública. Clientes API com Bearer válido continuam funcionando sem cookie. Bearer inválido não cai silenciosamente para a sessão.
- Access tokens são validados a cada requisição e renovados no backend quando expirados. Falha na renovação revoga a sessão.
- O botão Sign out apaga a sessão local e encaminha o navegador ao endpoint de logout do Keycloak.
- Logout por backchannel valida o JWT do provedor e bloqueia replay por jti. Revoga as sessões correspondentes ao sid/subject e cancela streams ativos.
- SSE reconecta no máximo na expiração do access token, passando novamente pela autenticação e atualização das permissões. A sessão local possui também um limite absoluto, de no máximo 24 horas.
- Mudanças de grupos/roles entram em vigor quando um token novo é emitido. A desativação no IdP impede renovação; logout administrativo com backchannel encerra as sessões locais sem esperar a expiração. Não é feita introspecção a cada chamada.

### Limite operacional atual

As sessões, transações de login e IDs de logout recentes estão em memória, com limites de capacidade e expiração. Use **uma réplica da API**. Reiniciar a API exige novo login. Antes de operar múltiplas réplicas, implemente um store compartilhado de sessões e distribuição de revogação; afinidade sozinha não garante o backchannel em todas as instâncias. Não coloque as credenciais de sessão em cookies assinados com uma chave padrão.

A API também continua aceitando Bearer OIDC sem habilitar login web: basta omitir `CROSSWATC_OIDC_CLIENT_ID`, mantendo issuer e audience configurados, para ambientes com um gateway de identidade externo.

## Validação

Os testes incluem OIDC simulado com JWKS assinado, PKCE, state incorreto/ausente/expirado/reutilizado, ID Token inválido, nonce e subject divergentes, refresh, CSRF, roles, logout e cancelamento de streams, além da preservação de redirect e cookies no proxy web.

```sh
go test -race ./internal/auth ./internal/config ./internal/httpapi
pnpm --filter web typecheck
pnpm --filter web lint
pnpm --filter web test
```

Validação local em 21/10/2026: login real no Keycloak, retorno ao dashboard com `crosswatc-dev`, acesso à administração pelas roles e logout com retorno à tela de acesso. O Keycloak enviou backchannel logout aceito com HTTP 200; após a saída, `/api/v1/me` retornou 401. Build Docker, suíte completa de Go, testes de concorrência da autenticação/API, TypeScript, lint, 54 testes web e lint/render do Helm passaram.

## Helm

Crie previamente um Secret com a chave `client-secret` e configure os valores abaixo. O chart exige uma réplica da API e usa estratégia Recreate quando o login web está habilitado.

```yaml
api:
  replicas: 1
auth:
  mode: oidc
  oidcIssuer: https://identity.example.com/realms/crosswatc
  oidcAudience: crosswatc-api
  rolesClientID: crosswatc-api
  adminGroup: platform-admin
  browser:
    clientID: crosswatc-web
    publicURL: https://workspace.example.com
    existingSecret: crosswatc-oidc
```
