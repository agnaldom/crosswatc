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
