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

Requisitos: Go ≥ 1.25, Node 22, pnpm, Docker.

```bash
make db-up            # PostgreSQL local
make build test       # Go
pnpm install
pnpm dev              # UI em http://localhost:3000
```
