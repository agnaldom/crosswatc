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

Requisitos: Go ≥ 1.26, Node 22, pnpm, Docker.

```bash
make db-up            # PostgreSQL local
make build test       # Go
pnpm install
pnpm dev              # UI em http://localhost:3000
```

Rodando a stack local sem cluster (dados de exemplo):

```bash
psql "$DATABASE_URL" -f db/seeds/dev.sql   # após o worker aplicar as migrations
CROSSWATC_AUTH_MODE=dev CROSSWATC_DATABASE_URL=$DATABASE_URL go run ./services/api
CROSSWATC_API_URL=http://localhost:8080 CROSSWATC_DEV_USER=agnaldo CROSSWATC_DEV_SQUADS=payments pnpm dev
```

`CROSSWATC_DEV_*` só funciona com a API em `CROSSWATC_AUTH_MODE=dev`, que é recusado quando `CROSSWATC_ENV=production`.
