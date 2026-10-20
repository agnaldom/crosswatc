<p align="center">
  <img src="apps/web/public/logo.svg" alt="Crosswatc Logo" width="380" />
</p>

# crosswatc

Developer Experience e Lifecycle Management sobre GitOps (GitHub → Argo CD → Crossplane → Cloud).

Spec: [`crosswatc-spec-v4.md`](crosswatc-spec-v4.md) · Checklist: §69.

## Layout

| Caminho | Conteúdo |
|---|---|
| `apps/web` | Next.js UI |
| `services/api` | Go API (Gin) |
| `services/worker` | Go Worker (discovery, adapters, reconciler) |
| `internal/` | Pacotes Go compartilhados |
| `db/migrations` | Migrations PostgreSQL |
| `deploy/` | Helm / manifests |

## Desenvolvimento

Requisitos: Go ≥ 1.26, Node 22, pnpm, Docker (ou um PostgreSQL ≥ 15 próprio).

```bash
make build test       # compila e testa o backend Go
pnpm install          # dependências da UI
```

## Rodando tudo com Docker Compose

```bash
docker compose up                    # constrói as imagens na primeira execução e sobe toda a aplicação
# ou, em segundo plano, aguardando a aplicação ficar pronta:
docker compose up -d --build --wait
```

Sobe `apps/web`, API, worker e PostgreSQL. O serviço `migrate` aplica as migrations antes dos
serviços Go; `kubeconfig` prepara o acesso ao Kubernetes local. O Compose usa o cluster existente
`kind-crossplane-poc`, com Crossplane instalado, pela rede Docker `kind`. O cluster precisa estar
rodando. O contexto atual do seu `kubectl` não é alterado.

As portas são escolhidas automaticamente dentro de faixas para evitar conflitos com outros
projetos: web `3000–3099`, API `8080–8099`, worker `8181–8199` e PostgreSQL `5432–5499`.
A comunicação entre os contêineres usa os nomes dos serviços e independe da porta escolhida.
Consulte os endereços publicados:

```bash
docker compose ps
docker compose port web 3000
docker compose port api 8080
docker compose port postgres 5432
docker compose logs -f api worker web
docker compose down                  # para os serviços, preservando o banco e o cluster existente
```

Para fixar uma porta livre, use `WEB_PORT=3100 docker compose up -d` (também existem
`API_PORT`, `WORKER_PORT` e `POSTGRES_PORT`). Uma porta fixa ocupada causa erro;
remova a variável para voltar à seleção automática. As portas são publicadas apenas em localhost.

Para outro cluster local, configure as variáveis em `.env`:

```dotenv
CROSSWATC_KUBECONFIG=/caminho/absoluto/kubeconfig
KUBE_CONTEXT=kind-crossplane-poc
KUBE_SERVER=https://crossplane-poc-control-plane:6443
KUBE_DOCKER_NETWORK=kind
```

`KUBE_SERVER` deve ser acessível pela rede Docker indicada e estar coberto pelo certificado do
cluster. O kubeconfig do contexto selecionado deve conter credenciais utilizáveis no contêiner
(sem plugins de autenticação externos). O arquivo original é montado somente para leitura.

O ambiente usa autenticação dev com usuário `agnaldo`, squads `payments,cards` e acesso SRE.
O healthcheck da web também aguarda a API e o worker ficarem prontos. GitHub App e integrações
externas continuam dependendo das suas credenciais. O avaliador `services/policy` é uma CLI de CI,
disponível sob demanda, sem processo permanente:

```bash
docker compose run --build --rm -v "$PWD:/work:ro" policy -head /work/claim.yaml -name minha-claim
```

Depois de alterar o código, execute `docker compose up -d --build --wait` para reconstruir as imagens.

## Rodando o backend

O backend são dois binários que compartilham o PostgreSQL:

| Binário | O que faz | Portas |
|---|---|---|
| `crosswatc-api` (`services/api`) | API REST + SSE (Gin), webhooks GitHub/Argo | `:8080` API · `:9090` `/metrics` |
| `crosswatc-worker` (`services/worker`) | discovery no cluster, reconciler, informers, notificações; **aplica as migrations** | `:8081` `/healthz` `/readyz` `/metrics` |

### 1. PostgreSQL

```bash
make db-up                                     # docker compose: postgres em localhost:5432
export DATABASE_URL='postgres://crosswatc:crosswatc@localhost:5432/crosswatc?sslmode=disable'
```

### 2. Migrations (e dados de exemplo)

Em produção o worker aplica as migrations ao iniciar. Sem cluster, use o `dbtool`:

```bash
CROSSWATC_DATABASE_URL=$DATABASE_URL go run ./services/dbtool migrate
psql "$DATABASE_URL" -f db/seeds/dev.sql       # opcional: 3 recursos de exemplo (squads payments e cards)
```

### 3. API

Modo dev: a identidade vem dos headers `X-Dev-User` / `X-Dev-Squads` em vez de OIDC.

```bash
CROSSWATC_AUTH_MODE=dev CROSSWATC_LOG_FORMAT=text \
CROSSWATC_DATABASE_URL=$DATABASE_URL \
go run ./services/api
```

Testando:

```bash
curl localhost:8080/readyz
curl -H 'X-Dev-User: agnaldo' -H 'X-Dev-Squads: payments' localhost:8080/api/v1/resources
```

Sem `KUBECONFIG` a API sobe normalmente; só os endpoints de catálogo (`/catalog/*`, formulários e preview de mudanças) respondem `503`. Sem GitHub App, `POST /changes` e decommission respondem `503`.

### 4. Worker (precisa de um cluster)

O worker lê Claims/XRs/MRs do cluster atual do `KUBECONFIG` (Crossplane instalado; Argo CD opcional). Localmente, desligue a leader election:

```bash
KUBECONFIG=~/.kube/config \
CROSSWATC_DATABASE_URL=$DATABASE_URL \
CROSSWATC_LEADER_ELECTION=false CROSSWATC_LOG_FORMAT=text \
go run ./services/worker
```

Sem `KUBECONFIG` (fora de um pod) o worker para com `kube: in-cluster config (set KUBECONFIG for local development)`.

Para acompanhar PRs, CI e reviews, configure o GitHub App no worker e na API:

```bash
export CROSSWATC_GITHUB_APP_ID=123456 CROSSWATC_GITHUB_INSTALLATION_ID=7890123 \
       CROSSWATC_GITHUB_PRIVATE_KEY_PATH=$HOME/.config/crosswatc/github-app.pem
```

### 5. UI apontando para a API local

```bash
CROSSWATC_API_URL=http://localhost:8080 CROSSWATC_DEV_USER=agnaldo CROSSWATC_DEV_SQUADS=payments pnpm dev
# http://localhost:3000
```

### Variáveis principais

| Variável | Default | Uso |
|---|---|---|
| `CROSSWATC_DATABASE_URL` | — | PostgreSQL (obrigatória) |
| `CROSSWATC_ENV` | `development` | `production` proíbe `AUTH_MODE=dev` |
| `CROSSWATC_AUTH_MODE` | `oidc` | `oidc` ou `dev` |
| `CROSSWATC_OIDC_ISSUER` / `_AUDIENCE` | — | obrigatórias com `oidc` em produção |
| `KUBECONFIG` | — | vazio = configuração in-cluster |
| `CROSSWATC_GITHUB_APP_ID` / `_INSTALLATION_ID` / `_PRIVATE_KEY_PATH` | — | GitHub App (PRs, mudanças, drift) |
| `CROSSWATC_GITHUB_WEBHOOK_SECRET` / `CROSSWATC_ARGOCD_WEBHOOK_TOKEN` | — | habilitam `/webhooks/*` |
| `CROSSWATC_LEADER_ELECTION` | `true` | `false` para rodar o worker fora do cluster |
| `CROSSWATC_RESYNC_INTERVAL` | `5m` | resync completo do worker |
| `CROSSWATC_HTTP_ADDR` / `CROSSWATC_METRICS_ADDR` | `:8080` / `:9090` | portas da API |
| `CROSSWATC_LOG_FORMAT` / `_LEVEL` | `json` / `info` | `text` é mais legível localmente |

Lista completa em [`internal/config/config.go`](internal/config/config.go). Deploy no cluster: [`deploy/helm/crosswatc`](deploy/helm/crosswatc/README.md).

### Testes

```bash
make test                                       # unitários
make test-integration DATABASE_URL=$DATABASE_URL  # integração com PostgreSQL
make lint
```
