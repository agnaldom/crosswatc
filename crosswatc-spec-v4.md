# SPEC — Crosswatc

**Versão:** 4.0  
**Data:** 2026-10-17  
**Status:** Aprovada para desenvolvimento (Fundação + Fase 1)  
**Arquitetura:** Single Kubernetes Cluster  
**Stack principal:** Next.js + Go + PostgreSQL + Kubernetes + Argo CD + Crossplane + GitHub

---

## 1. Visão Geral

O **Crosswatc** será a camada de Developer Experience e Lifecycle Management sobre o fluxo GitOps já existente:

```text
Backstage Portal
        ↓
GitHub Pull Request
        ↓
CI / Policies / Approval
        ↓
Merge
        ↓
Argo CD
        ↓
Kubernetes
        ↓
Crossplane
        ↓
Cloud
```

O objetivo é permitir que o solicitante acompanhe o processo do início ao fim, visualize estado real, entenda falhas, saiba quem deve agir, consulte recursos já provisionados, edite parâmetros autorizados e acompanhe atualizações com histórico e auditoria completos.

---

## 2. Objetivos

A plataforma deve permitir:

1. acompanhar cada solicitação ponta a ponta;
2. derivar status das ferramentas reais;
3. traduzir estados técnicos para linguagem de negócio;
4. mostrar detalhes técnicos sob demanda;
5. indicar quem precisa agir;
6. consultar infraestrutura já provisionada;
7. editar somente parâmetros autorizados;
8. gerar preview e diff antes de mudanças;
9. gerar Pull Request automaticamente;
10. acompanhar updates em tempo real;
11. manter histórico de revisions;
12. manter histórico de operations;
13. detectar drift;
14. visualizar Claim → XR → Managed Resources;
15. manter auditoria completa.

---

## 3. Princípios Arquiteturais

### 3.1 Git é a Source of Truth

Toda alteração deve passar por Git:

```text
UI
 ↓
Go API
 ↓
GitHub
 ↓
Branch
 ↓
Commit
 ↓
Pull Request
 ↓
CI / Approval
 ↓
Merge
 ↓
Argo CD
 ↓
Crossplane
 ↓
Cloud
```

A plataforma **não altera diretamente**:

- Cloud Provider;
- Claims;
- XRs;
- Managed Resources;
- Argo CD Applications.

### 3.2 Estados

```text
Git                = Desired State
Kubernetes Claim   = Applied Desired State
XR + MR            = Observed Infrastructure State
Cloud              = Real Infrastructure
```

### 3.3 Estado derivado

A UI nunca grava manualmente a etapa atual. O backend resolve o estado com base em sinais reais.

### 3.4 Single Cluster

A arquitetura será **single-cluster**.

Não implementar:

- cluster selector;
- context selector;
- multi-kubeconfig;
- cluster registry;
- multi-cluster routing;
- multi-cluster cache.

Produção:

```text
Go Worker
   ↓
ServiceAccount
   ↓
Kubernetes API
```

Desenvolvimento:

```text
Go Worker
   ↓
KUBECONFIG
   ↓
Mesmo cluster de controle
```

---

## 4. Escopo

### Incluído

- lifecycle do provisionamento;
- lifecycle de updates;
- GitHub PR;
- CI/approval;
- Argo CD;
- Crossplane;
- Claim/XR/MR;
- topology;
- events;
- health;
- diff;
- revisions;
- operations;
- dynamic forms;
- drift;
- audit;
- SSE.

### Fora de escopo inicial

- multi-cluster;
- CRUD direto no Kubernetes;
- edição direta na Cloud;
- console administrativo completo de Argo CD;
- console administrativo completo de Crossplane;
- decomissionamento no primeiro MVP.

---

## 5. Stack Técnica

### Frontend

```text
Next.js
TypeScript
React
Tailwind CSS
TanStack Query
React Hook Form
Zod
XYFlow / React Flow
```

### Backend

```text
Go
Gin
client-go
dynamic client
informers
OpenTelemetry
```

### Banco

```text
PostgreSQL
```

### Infraestrutura

```text
Kubernetes
Argo CD
Crossplane
GitHub
```

### Realtime

```text
Server-Sent Events
```

---

## 6. Arquitetura de Alto Nível

```text
                    ┌───────────────────────┐
                    │   Developer / Squad   │
                    └──────────┬────────────┘
                               │
                               ▼
                    ┌───────────────────────┐
                    │      Next.js UI       │
                    │      Crosswatc        │
                    └──────────┬────────────┘
                               │
                         REST + SSE
                               │
                    ┌──────────▼────────────┐
                    │        Go API         │
                    │                      │
                    │ AuthN / AuthZ        │
                    │ Resource API         │
                    │ Revision API         │
                    │ Operations API       │
                    │ Change API           │
                    │ Catalog API          │
                    └──────────┬────────────┘
                               │
                         PostgreSQL
                               ▲
                               │
                    ┌──────────┴────────────┐
                    │      Go Worker        │
                    │                      │
                    │ Informer Manager     │
                    │ Discovery Engine     │
                    │ Event Normalizer     │
                    │ State Reconciler     │
                    │ GitOps Orchestrator  │
                    └─────┬────┬────┬──────┘
                          │    │    │
                          ▼    ▼    ▼
                       GitHub Argo K8s API
                                CD    │
                                      ▼
                                  Crossplane
                                      │
                                      ▼
                                    Cloud
```

---

## 7. Deployments

Inicialmente:

```text
crosswatc-web
crosswatc-api
crosswatc-worker
postgresql
```

Opcional:

```text
redis
```

---

## 8. Identidade e Naming

O nome oficial do projeto e do repositório é:

```text
crosswatc
```

Regras obrigatórias de naming:

- nenhum componente deve usar nomes, labels, annotations, métricas ou referências com nomes legados/anteriores do projeto;
- prefixo padrão de recursos internos: `crosswatc`;
- domínio recomendado para labels/annotations Kubernetes: `crosswatc.io`;
- correlation IDs devem usar prefixo `crosswatc-req-`;
- nomes de deployments devem usar `crosswatc-*`;
- métricas Prometheus devem usar prefixo `crosswatc_*`.

---

## 9. Repositório

Nome do repositório:

```text
crosswatc
```

Estrutura recomendada:

```text
crosswatc/
├── apps/
│   └── web/                         # Next.js
├── services/
│   ├── api/                         # Go API
│   └── worker/                      # Go Worker
├── packages/
│   ├── contracts/
│   └── ui-components/
├── internal/                        # código Go compartilhado (go.mod único na raiz)
│   ├── config/
│   ├── domain/
│   ├── lifecycle/
│   ├── gitops/
│   ├── events/
│   ├── policy/
│   ├── kube/                        # client single-cluster, dynamic, discovery, informers
│   ├── topology/
│   ├── store/                       # PostgreSQL (pgx)
│   ├── httpapi/                     # handlers Gin
│   └── adapters/                    # github, argocd, backstage
├── db/
│   ├── migrations/
│   └── seeds/
├── deploy/
│   ├── helm/
│   └── kubernetes/
├── docs/
│   ├── architecture/
│   ├── adr/
│   └── runbooks/
├── Makefile
├── docker-compose.yml
└── README.md
```

---

## 10. Bounded Contexts

```text
Resource
Revision
Operation
Lifecycle
GitOps
Crossplane
Topology
Events
Catalog
Policies
Audit
Drift
```

Esses contextos serão módulos internos inicialmente, não microservices separados.

---

## 11. TrackedResource

```typescript
interface TrackedResource {
  id: string
  name: string
  kind: string
  provider: 'azure' | 'aws' | 'huawei'
  environment: string
  owner: string
  requestedBy?: string
  region?: string
  resourceGroup?: string

  claimRef: {
    namespace: string
    name: string
    kind: string
    apiVersion: string
  }

  repository: string
  gitPath: string

  resourceState: ResourceState
  currentRevision: number

  composition?: {
    name: string
    revision?: string
  }

  createdAt: string
  updatedAt: string
}
```

---

## 12. Resource State

```typescript
type ResourceState =
  | 'PROVISIONING'
  | 'OPERATIONAL'
  | 'UPDATING'
  | 'DEGRADED'
  | 'DELETE_PENDING'
  | 'DELETING'
  | 'DELETED'
  | 'FAILED'    // provisionamento inicial falhou (nunca ficou OPERATIONAL)
  | 'UNKNOWN'   // sinais insuficientes (ex.: Claim não encontrado, informer desconectado)
```

Regras de derivação (avaliadas em ordem, primeira que casar vence):

| Condição | ResourceState |
|---|---|
| Claim com `deletionTimestamp` | `DELETING` |
| Revision `DELETE` aberta (não terminal) | `DELETE_PENDING` |
| Claim ausente e Revision `DELETE` `SUCCEEDED` | `DELETED` |
| Claim ausente e sem Revision `DELETE` | `UNKNOWN` |
| Nunca ficou operacional e Revision `CREATE` `FAILED` | `FAILED` |
| Claim, XR e todos MRs `Ready=True` e `Synced=True`, com Revision `UPDATE`/`PLATFORM_UPGRADE` aberta | `UPDATING` |
| Claim, XR e todos MRs `Ready=True` e `Synced=True` | `OPERATIONAL` |
| Já foi operacional e algum `Ready=False`/`Synced=False` | `DEGRADED` (ou `UPDATING` se houver Revision de update aberta) |
| Nunca ficou operacional | `PROVISIONING` |

O estado do recurso é separado do estado da mudança.

Exemplo:

```text
Resource: OPERATIONAL
Revision 8: WAITING_APPROVAL
```

---

## 13. Revision

```typescript
interface Revision {
  id: string
  resourceId: string
  rev: number

  kind:
    | 'CREATE'
    | 'UPDATE'
    | 'DELETE'
    | 'PLATFORM_UPGRADE'

  changes: SpecChange[]

  branch?: string
  commitSha?: string

  prNumber?: number
  prUrl?: string

  argoApp?: string
  state: RevisionState

  startedAt: string
  completedAt?: string
}
```

---

## 14. Revision State

```typescript
type RevisionState =
  | 'CREATED'
  | 'VALIDATING'
  | 'WAITING_APPROVAL'
  | 'MERGED'
  | 'SYNCING'
  | 'RECONCILING'
  | 'APPLYING'
  | 'SUCCEEDED'
  | 'FAILED'
  | 'REJECTED'    // PR fechado sem merge ou review "changes requested" final
  | 'CANCELLED'   // cancelado pelo solicitante antes do merge
  | 'SUPERSEDED'  // substituída por revision mais nova antes do merge
```

Estados terminais: `SUCCEEDED`, `FAILED`, `REJECTED`, `CANCELLED`, `SUPERSEDED`.

`failureReason` (ver §51) é obrigatório quando `state ∈ {FAILED, REJECTED}`.

### 14.1 Concorrência

- No máximo **uma** Revision não terminal por recurso.
- `POST /changes` com Revision aberta retorna `409 Conflict` com `revisionId` aberta.
- Garantido por índice único parcial no PostgreSQL (§37).

---

## 15. Operation

```typescript
interface Operation {
  id: string
  resourceId: string
  revisionId?: string
  correlationId: string

  type:
    | 'CREATE'
    | 'UPDATE'
    | 'DELETE'
    | 'PLATFORM_UPGRADE'
    | 'RECONCILE'

  status:
    | 'QUEUED'
    | 'RUNNING'
    | 'WAITING_APPROVAL'
    | 'SUCCEEDED'
    | 'FAILED'
    | 'CANCELLED'

  requestedBy: string
  requestedAt: string
  startedAt?: string
  finishedAt?: string
}
```

---

## 16. Correlation ID

Formato:

```text
crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T
```

Regra: `crosswatc-req-` + ULID (26 chars, Crockford base32, maiúsculo). Regex: `^crosswatc-req-[0-9A-HJKMNP-TV-Z]{26}$`.

Para recursos descobertos sem annotation de request (criados fora do Crosswatc), o worker gera um correlation ID novo na primeira descoberta.

Deve aparecer em:

- Operation;
- Revision;
- Events;
- Logs;
- Traces;
- Pull Request;
- Claim annotations.

---

## 17. Contrato de Labels e Annotations

```yaml
metadata:
  labels:
    crosswatc.io/managed: "true"
    crosswatc.io/owner: payments
    crosswatc.io/environment: dev
    crosswatc.io/provider: azure
    crosswatc.io/resource-type: postgresql

  annotations:
    crosswatc.io/request-id: crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T
    crosswatc.io/revision: "4"
    crosswatc.io/repository: crosswatc-org/argocd
    crosswatc.io/path: claims/dev/pgsql-payments.yaml
    crosswatc.io/argo-app: claims-dev            # opcional; senão resolvido via tracking-id do Argo CD
    crosswatc.io/backstage-task-id: 6f1c...      # opcional; habilita Etapa 1 (§31)
```

Labels obrigatórias: `managed`, `owner`, `environment`, `provider`, `resource-type`.
Annotations obrigatórias: `request-id`, `repository`, `path`.
Claims com `managed=true` sem campos obrigatórios aparecem com `ResourceState = UNKNOWN` e evento `INFRA.CLAIM.INVALID_METADATA`.

---

## 18. Desired / Applied / Observed State

```typescript
interface DesiredState {
  repository: string
  path: string
  commitSha: string
  spec: Record<string, unknown>
}

interface AppliedState {
  resourceVersion?: string
  generation?: number
  spec: Record<string, unknown>
  syncedWithGit: boolean
}

interface ObservedState {
  ready: boolean
  synced: boolean
  spec?: Record<string, unknown>
  lastReconcileAt?: string
}
```

---

## 19. Drift

```typescript
interface DriftState {
  detected: boolean

  differences?: {
    field: string
    desired: unknown
    observed: unknown
  }[]

  detectedAt?: string
}
```

---

## 20. Crossplane Discovery Engine

O Worker deve descobrir dinamicamente:

```text
XRDs
Claims
XRs
MRDs
Managed Resources
Compositions
CompositionRevisions
Relationships
```

A implementação deve aproveitar a abordagem usada por projetos como Crossview, porém sem multi-cluster.

---

## 21. Dynamic Kubernetes Client

Utilizar:

```go
dynamic.NewForConfig(config)
```

Modelo:

```go
type ResourceTarget struct {
    APIVersion string
    Kind       string
    Plural     string
}
```

Evitar clients hardcoded por provider.

---

## 22. XRD Discovery

Extrair:

```text
spec.group
spec.names.kind
spec.names.plural
spec.claimNames.kind
spec.claimNames.plural
spec.versions
OpenAPI schema
```

Uso:

- identificar APIs da plataforma;
- descobrir Claims;
- gerar Dynamic Forms;
- resolver relacionamentos;
- validar schemas.

---

## 23. MRD Discovery

Extrair:

```text
group
versions
kind
plural
```

Objetivo:

```text
descobrir Managed Resources dinamicamente
```

Sem hardcode por provider.

---

## 24. Paginação Kubernetes

Toda listagem deve utilizar:

```text
limit
continueToken
```

Exemplo inicial:

```text
limit = 500
```

Nunca carregar o cluster inteiro em uma única request.

---

## 25. Informer Manager

Como há apenas um cluster:

```text
Informer Manager
      ├── Claims
      ├── XRs
      ├── Managed Resources
      ├── Compositions
      ├── CompositionRevisions
      ├── Secrets metadata
      └── Kubernetes Events
```

Eventos:

```text
ADD
UPDATE
DELETE
```

Fluxo:

```text
Informer
   ↓
Raw Kubernetes Event
   ↓
Event Normalizer
   ↓
Canonical Event
   ↓
State Reconciler
   ↓
PostgreSQL
   ↓
SSE
   ↓
UI
```

---

## 26. Kubernetes Access

### Produção

```go
rest.InClusterConfig()
```

Usando:

```text
ServiceAccount + RBAC
```

### Local

```text
KUBECONFIG
```

Não implementar context switch.

---

## 27. RBAC

Read-only para:

```text
XRDs
Claims
XRs
Compositions
CompositionRevisions
ManagedResourceDefinitions
Managed Resources
Events
Secret metadata
```

Não conceder:

```text
create
update
patch
delete
```

para infraestrutura.

Também read-only: `applications.argoproj.io` (Argo CD, §35.1).

Única exceção de escrita: `coordination.k8s.io/leases` (get/create/update) **somente no namespace do Crosswatc** via `Role`, para leader election (§35.2).

---

## 28. Topology Resolver

Relacionamentos:

```text
Claim
  ↓
XR
  ↓
Managed Resources
```

Usar:

- resourceRef;
- resourceRefs;
- ownerReferences;
- composition-resource-name;
- labels;
- annotations;
- Crossplane references.

---

## 29. Topology API

```http
GET /api/v1/resources/{id}/topology
```

Response:

```json
{
  "nodes": [
    {
      "id": "claim/pgsql-payments",
      "type": "claim",
      "status": "ready"
    },
    {
      "id": "xr/xpostgresql-92a",
      "type": "composite",
      "status": "ready"
    },
    {
      "id": "mr/postgresql-server",
      "type": "managed-resource",
      "provider": "azure",
      "status": "ready"
    }
  ],
  "edges": [
    {
      "source": "claim/pgsql-payments",
      "target": "xr/xpostgresql-92a"
    }
  ]
}
```

---

## 30. Topology UI

Utilizar:

```text
XYFlow / React Flow
```

Exemplo:

```text
Claim
pgsql-payments
       │
       ▼
XR
xpostgresql-92a
       │
       ├─────────────┐
       ▼             ▼
PrivateEndpoint   PostgreSQL
                       │
                       ▼
                    Azure
```

---

## 31. Lifecycle — 7 Etapas

### Etapa 1 — Backstage Portal

Fonte: annotation `crosswatc.io/backstage-task-id` + polling de `GET {BACKSTAGE_URL}/api/scaffolder/v2/tasks/{taskId}` (intervalo 30s enquanto não terminal; para após `completed`/`failed`/`cancelled`).

Sem annotation: etapa marcada como `SKIPPED` (recurso criado fora do Portal) e a jornada começa na Etapa 2.

Saída:

```text
Scaffolder task completed
```

### Estado de cada etapa

```typescript
type StageStatus = 'PENDING' | 'IN_PROGRESS' | 'SUCCEEDED' | 'FAILED' | 'SKIPPED'

interface StageView {
  stage: 'REQUEST' | 'PULL_REQUEST' | 'CI_POLICIES' | 'ARGO_CD' | 'CROSSPLANE' | 'CLOUD' | 'OPERATIONAL'
  status: StageStatus
  actionOwner?: 'requester' | 'approver' | 'platform' | 'squad'  // quem precisa agir (objetivo 5)
  message?: string            // linguagem de negócio
  technicalDetail?: string    // sob demanda
  startedAt?: string
  finishedAt?: string
}
```

Regra: etapa N só fica `IN_PROGRESS`/`SUCCEEDED` se N-1 estiver `SUCCEEDED` ou `SKIPPED`. Primeira etapa `FAILED` interrompe a avaliação; etapas seguintes ficam `PENDING`.

### Etapa 2 — Git & PR

Saída:

```text
PR created
```

### Etapa 3 — CI & Policies

Validar:

- lint;
- schema;
- naming;
- tags;
- policies;
- CODEOWNERS;
- security.

Saída:

```text
checks success + approvals
```

### Etapa 4 — Argo CD

Saída:

```text
Sync = Synced
```

### Etapa 5 — Crossplane

Saída:

```text
XR Synced=True
```

### Etapa 6 — Cloud

Saída:

```text
todos MRs Ready=True
```

### Etapa 7 — Operational

Critérios:

```text
Claim Ready
XR Ready
MRs Ready
Secret metadata available
Catalog registered
```

Resultado:

```text
Resource State = OPERATIONAL
Revision State = SUCCEEDED
```

---

## 32. Revision Lifecycle

```text
CREATED
   ↓
VALIDATING
   ↓
WAITING_APPROVAL
   ↓
MERGED
   ↓
SYNCING
   ↓
RECONCILING
   ↓
APPLYING
   ↓
SUCCEEDED
```

Transições de saída (não happy path):

| De | Sinal | Para |
|---|---|---|
| `CREATED`, `VALIDATING`, `WAITING_APPROVAL` | PR fechado sem merge | `REJECTED` |
| `CREATED`, `VALIDATING`, `WAITING_APPROVAL` | usuário cancela | `CANCELLED` (fecha PR) |
| `CREATED`, `VALIDATING`, `WAITING_APPROVAL` | nova revision criada p/ mesmo recurso | `SUPERSEDED` |
| `VALIDATING` | check obrigatório `failure` | permanece `VALIDATING` com `failureReason` (re-run possível) |
| `SYNCING` | Argo `sync-failed` após retries | `FAILED` |
| `RECONCILING`, `APPLYING` | condition `Synced=False`/`Ready=False` com reason de erro por > `stuckTimeout` (default 30 min) | `FAILED` |
| qualquer não terminal | sem progresso por > `stuckTimeout` | permanece, emite `OPERATION.STUCK` + alerta |

Mapeamento de sinais → estado:

| RevisionState | Sinal de entrada |
|---|---|
| `VALIDATING` | PR aberto, checks pendentes |
| `WAITING_APPROVAL` | checks obrigatórios `success`, aprovações insuficientes |
| `MERGED` | PR `merged=true` |
| `SYNCING` | Argo `operationState.phase=Running` com `syncResult.revision == mergeCommitSha` |
| `RECONCILING` | Argo `Synced` no commit; Claim `generation` > `observedGeneration` ou XR `Synced` ≠ True |
| `APPLYING` | XR `Synced=True`; algum MR `Ready≠True` |
| `SUCCEEDED` | Claim, XR e todos MRs `Ready=True` no commit da revision |

Transições são monotônicas: o reconciler nunca regride uma revision (ex.: `APPLYING` → `SYNCING`), exceto via evento explícito de retry.

---

## 33. Canonical Events

```text
REQUEST.CREATED
REQUEST.SCAFFOLDER.COMPLETED
REQUEST.SCAFFOLDER.FAILED

GITOPS.PR.CREATED
GITOPS.PR.CLOSED
GITOPS.CHECK.SUCCEEDED
GITOPS.PR.APPROVED
GITOPS.PR.MERGED
GITOPS.CHECK.FAILED

GITOPS.SYNC.STARTED
GITOPS.SYNC.SUCCEEDED
GITOPS.SYNC.FAILED

INFRA.CLAIM.CREATED
INFRA.CLAIM.DELETED
INFRA.CLAIM.INVALID_METADATA
INFRA.XR.CREATED

INFRA.RECONCILE.STARTED
INFRA.RECONCILE.READY
INFRA.RECONCILE.FAILED

INFRA.MANAGED_RESOURCE.CREATED
INFRA.MANAGED_RESOURCE.READY
INFRA.MANAGED_RESOURCE.FAILED

LIFECYCLE.OPERATIONAL
LIFECYCLE.DEGRADED
LIFECYCLE.DRIFT.DETECTED
LIFECYCLE.DRIFT.RESOLVED

OPERATION.STUCK
```

Campo `level`: `info | warning | error`. Campo `status`: `success | failure | in_progress`.

---

## 34. Canonical Event

```json
{
  "eventId": "evt-81727",
  "correlationId": "crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T",
  "resourceId": "pgsql-payments-prd",
  "revisionId": "rev-7",
  "operationId": "op-722",
  "source": "crossplane",
  "type": "INFRA.RECONCILE.READY",
  "status": "success",
  "timestamp": "2026-10-17T10:37:42Z",
  "metadata": {
    "claim": "pgsql-payments-prd",
    "xr": "xpostgresql-29ad",
    "composition": "postgresql-azure-v3"
  }
}
```

---

## 35. Event Sources

### GitHub

```text
pull_request
check_run
review
push
```

### Argo CD

```text
sync-running
sync-succeeded
sync-failed
health-degraded
```

### Kubernetes / Crossplane

```text
Informers
```

### Backstage

```text
Scaffolder task state (polling, §31)
```

### 35.1 Ingestão

Endpoints (servidos pela API, enfileirados para o Worker via tabela `inbox` no PostgreSQL):

```http
POST /webhooks/github     # valida X-Hub-Signature-256 (HMAC SHA-256, secret do GitHub App)
POST /webhooks/argocd     # valida header Authorization: Bearer <token> (argocd-notifications)
```

Regras:

- assinatura inválida → `401`, nada persistido;
- idempotência: `X-GitHub-Delivery` / `eventId` gravado em `inbox.delivery_id UNIQUE`; duplicado → `202` sem reprocessar;
- webhooks são **gatilho**, não verdade: ao receber, o worker relê o estado na fonte (GitHub API / Argo Application CR) antes de reconciliar;
- resync periódico (default 5 min) cobre webhooks perdidos;
- Argo CD Application é lida via dynamic client (CRD `applications.argoproj.io`) no próprio cluster — não usar API HTTP do Argo no MVP.

### 35.2 Worker HA

- Worker roda com `replicas: 2` e **leader election** (`coordination.k8s.io/Lease`, nome `crosswatc-worker`). Só o líder roda informers, polling e reconciler.
- API é stateless e escalável horizontalmente.

---

## 36. State Reconciler

```text
event
 ↓
load current projection
 ↓
read required external state
 ↓
resolve new state
 ↓
compare old/new
 ↓
persist
 ↓
emit lifecycle event
 ↓
publish SSE
```

---

## 37. PostgreSQL

Tabelas:

```text
tracked_resource
revision
operation
tracker_event
resource_projection
resource_topology
ingest_checkpoint
inbox
audit_log
```

Migrations versionadas em `db/migrations` (goose, SQL puro). Toda tabela de estado usa `updated_at` + `version BIGINT` para optimistic locking no reconciler.

### tracked_resource

```sql
CREATE TABLE tracked_resource (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL,
  kind TEXT NOT NULL,
  provider TEXT NOT NULL,
  environment TEXT NOT NULL,
  owner TEXT NOT NULL,
  requested_by TEXT,
  region TEXT,
  resource_group TEXT,
  claim_namespace TEXT,
  claim_name TEXT,
  claim_kind TEXT,
  claim_api_version TEXT,
  repository TEXT,
  git_path TEXT,
  resource_state TEXT,
  current_revision INTEGER,
  composition_name TEXT,
  composition_revision TEXT,
  correlation_id TEXT NOT NULL,
  ever_operational BOOLEAN NOT NULL DEFAULT false,
  version BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (claim_namespace, claim_name, claim_kind),
  UNIQUE (environment, name)
);

CREATE INDEX tracked_resource_owner ON tracked_resource(owner);
```

### revision

```sql
CREATE TABLE revision (
  id UUID PRIMARY KEY,
  resource_id UUID REFERENCES tracked_resource(id),
  rev INTEGER NOT NULL,
  kind TEXT NOT NULL,
  state TEXT NOT NULL,
  changes JSONB,
  branch TEXT,
  commit_sha TEXT,
  pr_number INTEGER,
  pr_url TEXT,
  argo_app TEXT,
  failure_reason JSONB,
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  UNIQUE(resource_id, rev)
);

-- §14.1: no máximo uma revision não terminal por recurso
CREATE UNIQUE INDEX revision_one_open_per_resource
ON revision(resource_id)
WHERE state NOT IN ('SUCCEEDED','FAILED','REJECTED','CANCELLED','SUPERSEDED');
```

### operation

```sql
CREATE TABLE operation (
  id UUID PRIMARY KEY,
  resource_id UUID REFERENCES tracked_resource(id),
  revision_id UUID REFERENCES revision(id),
  correlation_id TEXT UNIQUE NOT NULL,
  type TEXT NOT NULL,
  status TEXT NOT NULL,
  requested_by TEXT NOT NULL,
  requested_at TIMESTAMPTZ NOT NULL,
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ
);
```

### tracker_event

```sql
CREATE TABLE tracker_event (
  id UUID PRIMARY KEY,
  dedupe_key TEXT UNIQUE NOT NULL,  -- source + id nativo (delivery id, k8s event uid+count, ...)
  resource_id UUID,
  revision_id UUID,
  operation_id UUID,
  correlation_id TEXT,
  source TEXT NOT NULL,
  type TEXT NOT NULL,
  level TEXT NOT NULL,
  message TEXT,
  raw_payload JSONB,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX tracker_event_resource_ts
ON tracker_event(resource_id, created_at DESC);
```

### resource_projection

```sql
CREATE TABLE resource_projection (
  resource_id UUID PRIMARY KEY,
  desired_state JSONB,
  applied_state JSONB,
  observed_state JSONB,
  drift_state JSONB,
  last_reconcile_at TIMESTAMPTZ
);
```

### resource_topology

```sql
CREATE TABLE resource_topology (
  resource_id UUID PRIMARY KEY,
  nodes JSONB NOT NULL,
  edges JSONB NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);
```

`resource_projection` também guarda `stages JSONB` (lista de `StageView`, §31) e `health JSONB`.

### ingest_checkpoint

```sql
CREATE TABLE ingest_checkpoint (
  source TEXT PRIMARY KEY,          -- 'github', 'argocd', 'backstage', 'k8s:<gvr>'
  cursor TEXT,                      -- resourceVersion, etag, timestamp
  last_success_at TIMESTAMPTZ,
  last_error TEXT,
  updated_at TIMESTAMPTZ NOT NULL
);
```

### inbox

```sql
CREATE TABLE inbox (
  id BIGSERIAL PRIMARY KEY,
  source TEXT NOT NULL,
  delivery_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  payload JSONB NOT NULL,
  received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  processed_at TIMESTAMPTZ,
  attempts INTEGER NOT NULL DEFAULT 0,
  last_error TEXT,
  UNIQUE (source, delivery_id)
);

CREATE INDEX inbox_pending ON inbox(received_at) WHERE processed_at IS NULL;
```

Worker consome com `SELECT ... FOR UPDATE SKIP LOCKED`.

### audit_log

```sql
CREATE TABLE audit_log (
  id BIGSERIAL PRIMARY KEY,
  actor TEXT NOT NULL,
  action TEXT NOT NULL,
  resource_id UUID,
  revision_id UUID,
  operation_id UUID,
  correlation_id TEXT,
  before JSONB,
  after JSONB,
  pr_url TEXT,
  commit_sha TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

`tracker_event` e `audit_log` são append-only: o role da aplicação recebe apenas `INSERT, SELECT` nessas tabelas (sem `UPDATE`/`DELETE`).

Retenção: `tracker_event.raw_payload` anulado após 90 dias por job de manutenção que usa role separado (`crosswatc_maintenance`), único com `UPDATE (raw_payload)`; demais colunas mantidas.

---

## 38. Dynamic Forms

Fluxo:

```text
XRD
 ↓
OpenAPI Schema
 ↓
Catalog Adapter
 ↓
UI Schema
 ↓
Dynamic Form
```

UX metadata opcional:

```yaml
spec.parameters.vmSize:
  displayName: VM Size
  editable: true
  order: 20
  safeInPlace: true
  approvalRequired:
    dev: false
    hlg: true
    prd: true
```

---

## 39. API

### Catalog

```http
GET /api/v1/catalog/resource-types
GET /api/v1/catalog/resource-types/{kind}
GET /api/v1/catalog/resource-types/{kind}/schema
```

### Resources

```http
GET /api/v1/resources
GET /api/v1/resources/{id}
GET /api/v1/resources/{id}/state
GET /api/v1/resources/{id}/health
GET /api/v1/resources/{id}/events
GET /api/v1/resources/{id}/revisions
GET /api/v1/resources/{id}/operations
GET /api/v1/resources/{id}/topology
```

Filtros:

```text
owner
environment
provider
kind
status
q          # busca por nome (prefixo)
```

### Convenções da API

- Paginação por cursor: `?limit=50&cursor=<opaque>` (max 200). Response: `{ "items": [...], "nextCursor": "..." | null }`.
- `{id}` aceita UUID do `tracked_resource`.
- Erros seguem RFC 9457 (`application/problem+json`):

```json
{
  "type": "https://crosswatc.io/errors/revision-open",
  "title": "Revision already open",
  "status": 409,
  "detail": "Resource pgsql-payments-prd already has revision 8 in WAITING_APPROVAL",
  "correlationId": "crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T"
}
```

- Toda response carrega header `X-Correlation-Id` (gerado se ausente na request).
- Health/probes: `GET /healthz` (liveness), `GET /readyz` (DB + informers sincronizados no worker).
- Contrato publicado em OpenAPI 3.1 (`packages/contracts/openapi.yaml`); tipos TS do frontend gerados a partir dele.

### GET /api/v1/resources/{id}/state

```json
{
  "resourceState": "OPERATIONAL",
  "currentRevision": { "rev": 8, "state": "WAITING_APPROVAL", "prUrl": "..." },
  "stages": [ { "stage": "REQUEST", "status": "SKIPPED" }, "..." ],
  "drift": { "detected": false },
  "lastReconcileAt": "2026-10-17T10:37:42Z"
}
```

---

## 40. Update Preview

```http
POST /api/v1/resources/{id}/changes/preview
```

Request:

```json
{
  "changes": {
    "vmSize": "Standard_D4as_v5"
  }
}
```

Response:

```json
{
  "valid": true,
  "changes": [
    {
      "field": "vmSize",
      "from": "Standard_D2as_v5",
      "to": "Standard_D4as_v5",
      "impact": "IN_PLACE"
    }
  ],
  "policy": {
    "result": "PASS"
  },
  "impact": {
    "replacementRequired": false,
    "possibleDowntime": false,
    "approvalRequired": true
  },
  "warnings": []
}
```

---

## 41. Executar Update

```http
POST /api/v1/resources/{id}/changes
```

Request:

```json
{
  "changes": {
    "vmSize": "Standard_D4as_v5"
  },
  "justification": "Aumento de capacidade."
}
```

Response:

```json
{
  "operationId": "op-722",
  "revisionId": "rev-8",
  "correlationId": "crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T",
  "prNumber": 173,
  "prUrl": "https://github.com/..."
}
```

---

## 42. Update Workflow

```text
User
 ↓
Edit
 ↓
Dynamic Form
 ↓
Preview
 ↓
Policy Validation
 ↓
Impact Analysis
 ↓
Confirm
 ↓
Operation
 ↓
Revision
 ↓
Git Branch
 ↓
Commit
 ↓
Pull Request
 ↓
CI
 ↓
Approval
 ↓
Merge
 ↓
Argo CD
 ↓
Crossplane
 ↓
Cloud
 ↓
Revision SUCCEEDED
 ↓
Resource OPERATIONAL
```

---

## 43. Update Consistency

Antes de abrir PR:

```text
Git desired state
       ↓
compare
       ↑
Kubernetes Claim
```

Se houver divergência:

```text
UPDATE BLOCKED
```

Até o Applied State sincronizar novamente.

---

## 44. GitOps Orchestrator

Funções:

```text
readManifest()
createBranch()
updateManifest()
commitManifest()
createPullRequest()
getPullRequest()
getChecks()
getReviews()
getDiff()
```

Autenticação:

```text
GitHub App
```

Branch:

```text
create/{kind}/{environment}/{resource}

update/{kind}/{environment}/{resource}/rev-{n}

upgrade/{kind}/{environment}/{resource}/rev-{n}
```

---

## 45. Realtime

Usar SSE:

```http
GET /api/v1/stream
```

Eventos:

```text
resource.updated
revision.updated
operation.updated
event.created
topology.updated
drift.detected
```

Exemplo:

```text
event: revision.updated

data: {
  "resourceId": "pgsql-payments-prd",
  "revision": 8,
  "state": "RECONCILING"
}
```

Polling fallback do frontend:

```text
15s
```

Regras SSE:

- autenticado (mesma sessão OIDC da API); eventos filtrados pela mesma regra de AuthZ (§49) — usuário só recebe eventos de recursos que pode ler;
- filtro opcional `?resourceId=<id>`;
- cada mensagem tem `id:` monotônico; reconexão com `Last-Event-ID` reenvia eventos do buffer (últimos 5 min);
- heartbeat (`: ping`) a cada 20s;
- fan-out entre réplicas da API via `LISTEN/NOTIFY` do PostgreSQL (Redis só se `NOTIFY` não escalar).

---

## 46. UI

Navegação:

```text
Infrastructure
├── My Infrastructure
├── Requests
├── Changes
├── Operations
└── Catalog
```

Detalhe:

```text
pgsql-payment-prd

● Operational

PostgreSQL
Azure
Production

[Edit Configuration]
[Open GitOps]

Overview
Configuration
Lifecycle
Topology
Changes
Events
Health
```

### Overview

```text
Crossplane        ✓ Ready
Argo CD           ✓ Synced
Cloud             ✓ Ready
Configuration     ✓ In Sync
Secrets           ✓ Available
Monitoring        ✓ Active
```

### Lifecycle

```text
✓ Request
✓ Pull Request
✓ CI & Policies
✓ Argo CD
✓ Crossplane
✓ Cloud
✓ Operational
```

### Topology

```text
Claim
  │
  ▼
XR
  │
  ├── MR
  ├── MR
  └── MR
```

### Changes

```text
REV 8

Resize VM

D2 → D4

Requested by
agnaldo

PR #173

✓ Completed
```

### Events

```text
07:31 Request received
07:31 Configuration validated
07:32 Pull Request #173 created
07:34 Pull Request approved
07:35 Argo CD sync started
07:35 Crossplane reconciliation started
07:37 Cloud resource updated
07:38 Resource operational
```

---

## 47. Drift Detection

Exemplo:

```text
⚠ Configuration Drift

Desired
SKU: D4

Observed
SKU: D8
```

Sinais:

```text
Argo OutOfSync
Git != Claim
XR Synced=False
MR Ready=False
Cloud manual change
```

---

## 48. Resource Discovery

Descobrir Claims com:

```text
crosswatc.io/managed=true
```

Se ainda não estiverem no PostgreSQL:

```text
Create TrackedResource
```

---

## 49. Segurança

### AuthN

```text
OIDC corporativo
```

### AuthZ

```text
resource.owner ∈ user.squads
```

`user.squads` vem do claim `groups` do token OIDC, mapeado por prefixo configurável (ex.: grupo `squad-payments` → squad `payments`).

SRE (grupo configurável, ex.: `platform-sre`):

```text
global read
```

Modo desenvolvimento (`CROSSWATC_AUTH_MODE=dev`): usuário e squads vêm de headers `X-Dev-User` / `X-Dev-Squads`. Proibido em produção — API recusa iniciar com `dev` se `CROSSWATC_ENV=production`.

### GitHub

Usar GitHub App.

### Kubernetes

ServiceAccount read-only.

### Secrets

Nunca retornar secret values.

Mostrar somente existência e nome.

---

## 50. Auditoria

Registrar:

```text
who
what
when
resource
revision
operation
correlationId
before
after
PR
commit
```

Eventos devem ser append-only.

---

## 51. Error Normalization

Modelo:

```text
technicalCode
userMessage
technicalMessage
recommendedAction
runbook
owner
```

Exemplo:

```json
{
  "technicalCode": "NicInUse",
  "userMessage": "A interface de rede ainda está associada a outro recurso.",
  "recommendedAction": "Aguarde a reconciliação automática ou acione Plataforma.",
  "owner": "platform",
  "runbook": "/runbooks/azure/nic-in-use"
}
```

---

## 52. Observabilidade

Usar:

```text
OpenTelemetry
Prometheus
Grafana
```

ou Datadog.

Métricas:

```text
crosswatc_resources_total
crosswatc_operations_total
crosswatc_operations_failed_total
crosswatc_revision_duration_seconds
crosswatc_stage_duration_seconds
crosswatc_event_ingestion_lag_seconds
crosswatc_sse_connections
crosswatc_drift_total
crosswatc_informer_restarts_total
```

Tracing attributes:

```text
resource.id
revision.id
operation.id
correlation.id
owner
environment
provider
kind
```

---

## 53. SLO

```text
API availability: 99.5%

GET /resources/{id}: p95 < 400ms

External event → UI: p95 < 3s

Reconciler lag: p95 < 10s
```

---

## 54. Alertas

```text
GitHub webhook failures
Argo notification failures
Kubernetes informer disconnected
Worker backlog
State reconciler lag
Operation stuck
SSE error rate
Drift spike
```

---

## 55. MVP — Fase 1

Read Only:

```text
Single Cluster Client
Dynamic Client
XRD Discovery
MRD Discovery
Claim Discovery
XR Discovery
MR Discovery
GitHub Adapter
Argo Adapter
State Machine
Resource API
Next.js UI
```

Critério:

```text
Um recurso real aparece corretamente do Git até Cloud.
```

---

## 56. MVP — Fase 2

Realtime:

```text
Informers
Canonical Events
PostgreSQL
SSE
Revisions
Operations
Topology
History
```

Critério:

```text
Mudança externa aparece na UI em menos de 3 segundos.
```

---

## 57. MVP — Fase 3

Edição:

```text
Azure VM
Azure Storage
DEV
```

Implementar:

```text
XRD-driven forms
Preview
Diff
Impact Analysis
Policy Validation (básica: schema + metadata `editable`/`approvalRequired`)
GitHub PR
Update Revision
Update Consistency (§43)
```

Policy Validation completa (OPA/Kyverno, tags, naming) fica na Fase 4.

---

## 58. MVP — Fase 4

Hardening:

```text
RBAC
Audit
Drift
SLO
Alerts
Dashboards
Policy Validation
Load Testing
```

---

## 59. Piloto

```text
1 cluster
1 squad
1 environment DEV
Azure

Resources:
- VM
- Storage
```

---

## 60. Roadmap

### Decommissioning

```text
Request Delete
 ↓
Impact Preview
 ↓
Double Confirmation
 ↓
PR remove Claim
 ↓
Approval
 ↓
Merge
 ↓
Argo CD
 ↓
Crossplane Delete
 ↓
Cloud Delete
 ↓
DELETED
```

### Costs

```text
current estimated cost
new estimated cost
monthly variation
```

### Notifications

```text
WAITING_APPROVAL
FAILED
OPERATIONAL
DRIFT_DETECTED
```

Destinos:

```text
Teams
Email
```

---

## 61. EPICs

```text
EPIC 01 — Single Cluster Kubernetes Client
EPIC 02 — Crossplane XRD Discovery
EPIC 03 — Crossplane MRD Discovery
EPIC 04 — Claim / XR / MR Discovery
EPIC 05 — Informer Manager
EPIC 06 — Crossplane Relationship Resolver
EPIC 07 — Topology API
EPIC 08 — Resource API
EPIC 09 — GitHub Adapter
EPIC 10 — Argo CD Adapter
EPIC 11 — Canonical Event Model
EPIC 12 — Lifecycle State Machine
EPIC 13 — Operations Model
EPIC 14 — PostgreSQL Read Model
EPIC 15 — Next.js UI
EPIC 16 — Event Timeline
EPIC 17 — SSE
EPIC 18 — Revision History
EPIC 19 — Dynamic Forms
EPIC 20 — Change Preview
EPIC 21 — GitOps Update
EPIC 22 — Drift Detection
EPIC 23 — Security / RBAC
EPIC 24 — Observability / SLO
```

---

## 62. ADRs

```text
ADR-001 — Git as Source of Truth
ADR-002 — Single Cluster Architecture
ADR-003 — Resource State vs Revision State
ADR-004 — Dynamic Kubernetes Client
ADR-005 — Informer-Based Resource Watching
ADR-006 — Canonical Event Model
ADR-007 — SSE over WebSocket
ADR-008 — PostgreSQL as Read Model
ADR-009 — GitHub App Authentication
ADR-010 — XRD-Driven Dynamic Forms
ADR-011 — No Direct Kubernetes Mutation
ADR-012 — Monorepo Architecture
```

---

## 63. Acceptance Criteria — Read Only

O MVP estará pronto quando:

1. backend conectar a um único cluster;
2. XRDs forem descobertos;
3. Claims forem descobertos;
4. XR for correlacionado ao Claim;
5. MRs forem correlacionados ao XR;
6. GitHub PR aparecer;
7. CI aparecer;
8. Argo CD Sync aparecer;
9. Crossplane Conditions aparecerem;
10. Cloud state aparecer via MR;
11. UI mostrar as 7 etapas;
12. topology funcionar;
13. events aparecerem;
14. usuário não precisar usar kubectl.

---

## 64. Acceptance Criteria — Update

Update estará pronto quando:

1. recurso estiver OPERATIONAL;
2. usuário clicar Edit;
3. form for derivado do schema;
4. somente campos permitidos forem editáveis;
5. preview mostrar before/after;
6. policies forem executadas;
7. impacto aparecer;
8. usuário confirmar;
9. Operation for criada;
10. Revision for criada;
11. branch for criada;
12. PR for criado;
13. CI executar;
14. aprovação ocorrer;
15. Argo sincronizar;
16. Crossplane reconciliar;
17. MRs atualizarem;
18. Revision virar SUCCEEDED;
19. Resource voltar a OPERATIONAL;
20. histórico ficar disponível.

---

## 65. Referência Crossview

O Crossview será usado apenas como referência técnica de implementação para:

```text
Dynamic Kubernetes Client
Informers
XRD/MRD discovery
Managed Resource discovery
Relationships
Topology
Kubernetes Events
```

O projeto **não será um fork do Crossview**.

O Crosswatc possui escopo maior:

```text
Backstage
+
GitHub
+
CI
+
Argo CD
+
Crossplane
+
Cloud
+
Changes
+
Operations
+
Audit
+
Lifecycle
```

Também não serão reaproveitados conceitos de multi-context ou multi-cluster.

---

## 66. Decisão SSE vs WebSocket

Apesar de ferramentas como Crossview utilizarem WebSocket, o Crosswatc utilizará:

```text
SSE
```

Motivo:

```text
backend → frontend
```

é o fluxo predominante.

Ações do usuário continuam via REST.

---

## 67. Resultado Esperado

```text
                  CROSSWATC

┌─────────────────────────────────────────────────────────┐
│ Developer Experience                                    │
│                                                         │
│ Resources | Changes | Operations | Audit               │
└──────────────────────────┬──────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│ Next.js + Go Platform API                               │
│                                                         │
│ Resource | Revision | Operation | Lifecycle | Topology │
└──────────────────────────┬──────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│ GitOps                                                   │
│                                                         │
│ GitHub → CI → Approval → Merge → Argo CD                │
└──────────────────────────┬──────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│ Single Kubernetes Control Cluster                        │
│                                                         │
│ Crossplane                                               │
│                                                         │
│ Claim → XR → Composition → Managed Resources            │
└──────────────────────────┬──────────────────────────────┘
                           │
                           ▼
                         Cloud
```

---

## 68. Princípios Finais

```text
1. Git é a source of truth.
2. Existe apenas um cluster de controle.
3. Kubernetes é leitura operacional, não caminho de escrita.
4. Informers fornecem estado em tempo real.
5. Crossplane Discovery é dinâmico.
6. XRD/MRD evitam hardcode por provider.
7. Claim → XR → MR deve ser visualizável.
8. Toda mudança vira Pull Request.
9. Resource State e Revision State são separados.
10. Toda Operation é auditável.
11. SSE atualiza a interface.
12. PostgreSQL mantém read model e histórico.
```

O resultado final não será apenas um dashboard Crossplane. Será o portal de lifecycle da infraestrutura da Crosswatc Platform.

---

## 69. Checklist de Implementação

Cada item vira uma issue no GitHub (label `phase:*`). Todo PR referencia a issue (`Closes #N`) e só é mergeado com CI verde.

Status: **todos os itens implementados** (PRs #34–#66, empilhados; marcados `[x]` quando o PR existe com CI verde). Itens do roadmap em §69.1.

### Fundação

- [x] **F0** (#1) — Revisão do spec v4 + checklist
- [x] **F1** (#2) — Scaffold do monorepo: `go.mod`, `Makefile`, pnpm workspace, Next.js em `apps/web`, `docker-compose.yml` (PostgreSQL), golangci-lint, CI GitHub Actions (Go test/lint + web lint/build)
- [x] **F2** (#3) — Config, logging estruturado (`slog`), Correlation ID (ULID, §16) e contrato de labels/annotations (§17) com validação
- [x] **F3** (#4) — Migrations PostgreSQL (§37, goose) + camada `store` (pgx) com testes de integração
- [x] **F4** (#5) — Modelo de domínio + Lifecycle State Machine: derivação de `ResourceState` (§12), transições de `RevisionState` (§32), resolver das 7 etapas (§31)

### Fase 1 — Read Only

- [x] **P1.1** (#6) — Single Cluster Kubernetes Client: in-cluster/KUBECONFIG, dynamic client, lister paginado (`limit`/`continue`) — EPIC 01
- [x] **P1.2** (#7) — XRD Discovery (§22) — EPIC 02
- [x] **P1.3** (#8) — MRD Discovery (§23) — EPIC 03
- [x] **P1.4** (#9) — Claim / XR / MR Discovery + Resource Discovery (§48) — EPIC 04
- [x] **P1.5** (#10) — Relationship Resolver + Topology (§28, §29) — EPIC 06, 07
- [x] **P1.6** (#11) — GitHub Adapter read-only (GitHub App; PR, checks, reviews) — EPIC 09
- [x] **P1.7** (#12) — Argo CD Adapter (Application CR via dynamic client) — EPIC 10
- [x] **P1.8** (#13) — Backstage Adapter (Scaffolder task polling, §31)
- [x] **P1.9** (#14) — Worker: State Reconciler (resync periódico → sinais → projeção → PostgreSQL) — EPIC 12
- [x] **P1.10** (#15) — Resource API (Gin): catálogo, resources, state, health, events, revisions, operations, topology; AuthN (OIDC + modo dev) e AuthZ por squad — EPIC 08
- [x] **P1.11** (#16) — Next.js UI read-only: lista, detalhe, overview, lifecycle 7 etapas, topology (React Flow), events — EPIC 15
- [x] **P1.12** (#17) — Deploy: Dockerfiles, Helm chart, RBAC read-only (§27), probes

### Fase 2 — Realtime

- [x] **P2.1** (#18) — Informer Manager + leader election (§25, §35.2) — EPIC 05
- [x] **P2.2** (#19) — Canonical Event Model + ingestão por webhooks/inbox (§33–35.1) — EPIC 11
- [x] **P2.3** (#20) — SSE com AuthZ, `Last-Event-ID`, `LISTEN/NOTIFY` (§45) — EPIC 17
- [x] **P2.4** (#21) — Revisions & Operations history — EPIC 13, 18
- [x] **P2.5** (#22) — Event Timeline realtime na UI — EPIC 16

### Fase 3 — Edição

- [x] **P3.1** (#23) — XRD-driven Dynamic Forms (§38) — EPIC 19
- [x] **P3.2** (#24) — Change Preview: diff, impacto, policy básica (§40) — EPIC 20
- [x] **P3.3** (#25) — GitOps Update Orchestrator: branch, commit, PR (§41, §44) — EPIC 21
- [x] **P3.4** (#26) — Update Consistency + concorrência de revisions (§14.1, §43)

### Fase 4 — Hardening

- [x] **P4.1** (#27) — Security: OIDC produção, RBAC fino, revisão de secrets — EPIC 23
- [x] **P4.2** (#28) — Auditoria (`audit_log`, §50)
- [x] **P4.3** (#29) — Drift Detection (§19, §47) — EPIC 22
- [x] **P4.4** (#30) — Observabilidade, SLOs, alertas, dashboards (§52–54) — EPIC 24
- [x] **P4.5** (#31) — Policy Validation completa (OPA/Kyverno)
- [x] **P4.6** (#32) — Load testing

### Transversal

- [x] **D1** (#33) — ADRs 001–012 em `docs/adr` (§62)

### 69.1 Roadmap (§60)

- [x] **R1** (#67) — Decommissioning com preview de impacto e dupla confirmação
- [x] **R2** (#68) — Notificações (Teams e e-mail): WAITING_APPROVAL, FAILED, OPERATIONAL, DRIFT_DETECTED
- [ ] **R3** (#69) — Estimativa de custos no preview (Azure Retail Prices)

---

## 70. Changelog v3 → v4

- Versão do cabeçalho corrigida (3.0 → 4.0) e numeração de seções corrigida (§9).
- §8: removida contradição na regra de naming (proibia o próprio prefixo `crosswatc`).
- Backend: decidido **Gin** (antes "Gin ou Chi").
- §12: `ResourceState` ganhou `FAILED`/`UNKNOWN` e tabela de derivação.
- §14: `RevisionState` ganhou `REJECTED`/`SUPERSEDED`, estados terminais e regra de concorrência (§14.1).
- §16: Correlation ID com ULID completo (26 chars) e regex.
- §17: labels/annotations obrigatórias, `argo-app`, `backstage-task-id`.
- §27: RBAC inclui Argo `Application` (read) e exceção de `Lease` para leader election.
- §31: Etapa 1 via annotation + polling da Scaffolder API; modelo `StageView` com "quem precisa agir".
- §32: transições de falha e mapeamento sinal → estado.
- §33: novos eventos canônicos (PR fechado, claim inválido, drift resolvido, operação travada).
- §35.1/§35.2: webhooks com assinatura, idempotência, inbox, resync; worker com leader election.
- §37: `tracked_resource` com unicidade por claim/ambiente; índice de revision aberta; DDL de `ingest_checkpoint`, `inbox`, `audit_log`; append-only reforçado por grants.
- §39: convenções de paginação, erros RFC 9457, `X-Correlation-Id`, probes, OpenAPI.
- §45: SSE com AuthZ, `Last-Event-ID`, heartbeat e fan-out via `LISTEN/NOTIFY`.
- §49: origem das squads (claim `groups`) e modo dev bloqueado em produção.
- §57: Policy Validation básica antecipada para a Fase 3.
- §69: checklist de implementação.
