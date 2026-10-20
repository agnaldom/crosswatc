import http from "node:http";

export const mockResources = [
  {
    id: "b2c1a4e0-9876-4abc-9def-1234567890ab",
    name: "db-payments-prod",
    kind: "XPostgreSQLInstance",
    group: "database.example.org",
    version: "v1alpha1",
    namespace: "default",
    environment: "prod",
    squad: "payments",
    owner: "payments",
    provider: "azure",
    region: "brazilsouth",
    resourceGroup: "rg-prod-crossplane-01",
    repository: "crosswatc-org/argocd",
    gitPath: "claims/prod/db-payments.yaml",
    correlationId: "crosswatc-req-01JCA950C40227DXJM556P8M5T",
    claimRef: { namespace: "payments", name: "payments-db-claim", kind: "PostgreSQLInstance", apiVersion: "database.example.org/v1alpha1" },
    composition: { name: "postgresql-azure-v3" },
    resourceRefsCount: 2,
    responsive: true,
    resourceState: "OPERATIONAL",
    currentRevision: 1,
    synced: true,
    ready: true,
    state: "healthy",
    createdAt: "2026-10-17T12:53:46Z",
    updatedAt: "2026-10-19T18:00:00Z",
    conditions: [
      {
        type: "Responsive",
        status: "True",
        reason: "WatchCircuitClosed",
        message: "Resource informer active and responsive in cluster control plane.",
        lastTransitionTime: "2026-10-19T18:00:00Z",
      },
      {
        type: "Synced",
        status: "True",
        reason: "ReconcileSuccess",
        message: "Successfully reconciled composite resource.",
        lastTransitionTime: "2026-10-19T18:00:00Z",
      },
      {
        type: "Ready",
        status: "True",
        reason: "Available",
        message: "Resource is available and all managed resources are ready.",
        lastTransitionTime: "2026-10-19T18:00:00Z",
      },
    ],
  },
  {
    id: "c3d2e1f0-1122-3344-5566-778899aabbcc",
    name: "redis-session-cache",
    kind: "XRedisCluster",
    group: "cache.example.org",
    version: "v1alpha1",
    namespace: "default",
    environment: "staging",
    squad: "platform",
    owner: "platform",
    provider: "azure",
    region: "eastus2",
    resourceGroup: "rg-staging-crossplane-01",
    repository: "crosswatc-org/argocd",
    gitPath: "claims/staging/redis-session.yaml",
    correlationId: "crosswatc-req-02JCA950C40227DXJM556P8M5T",
    claimRef: { namespace: "platform", name: "redis-cache-claim", kind: "RedisCluster", apiVersion: "cache.example.org/v1alpha1" },
    composition: { name: "redis-azure-v2" },
    resourceRefsCount: 1,
    responsive: true,
    resourceState: "PROVISIONING",
    currentRevision: 1,
    synced: true,
    ready: false,
    state: "progressing",
    createdAt: "2026-10-18T10:00:00Z",
    updatedAt: "2026-10-19T18:10:00Z",
    conditions: [
      {
        type: "Responsive",
        status: "True",
        reason: "WatchCircuitClosed",
        message: "Resource informer active and responsive.",
        lastTransitionTime: "2026-10-19T18:10:00Z",
      },
      {
        type: "Synced",
        status: "True",
        reason: "ReconcileSuccess",
        message: "Applied manifests.",
        lastTransitionTime: "2026-10-19T18:10:00Z",
      },
      {
        type: "Ready",
        status: "False",
        reason: "CreatingOrPending",
        message: "Underlying Redis nodes are provisioning.",
        lastTransitionTime: "2026-10-19T18:10:00Z",
      },
    ],
  },
  {
    id: "d4e3f2a1-9988-7766-5544-33221100ffee",
    name: "storage-assets-bucket",
    kind: "XBucket",
    group: "storage.example.org",
    version: "v1alpha1",
    namespace: "default",
    environment: "prod",
    squad: "platform",
    owner: "platform",
    provider: "azure",
    region: "eastus2",
    resourceGroup: "rg-prod-crossplane-01",
    repository: "crosswatc-org/argocd",
    gitPath: "claims/prod/storage-assets.yaml",
    correlationId: "crosswatc-req-03JCA950C40227DXJM556P8M5T",
    claimRef: { namespace: "platform", name: "storage-bucket-claim", kind: "Bucket", apiVersion: "storage.example.org/v1alpha1" },
    composition: { name: "bucket-azure-v1" },
    resourceRefsCount: 1,
    responsive: true,
    resourceState: "DEGRADED",
    currentRevision: 1,
    synced: false,
    ready: false,
    state: "drifted",
    createdAt: "2026-10-14T12:53:45Z",
    updatedAt: "2026-10-19T18:15:00Z",
    conditions: [
      {
        type: "Responsive",
        status: "True",
        reason: "WatchCircuitClosed",
        message: "Resource informer active.",
        lastTransitionTime: "2026-10-19T18:15:00Z",
      },
      {
        type: "Synced",
        status: "False",
        reason: "ReconcilePending",
        message: "Cannot compose resources: bucket permissions pending.",
        lastTransitionTime: "2026-10-19T18:15:00Z",
      },
      {
        type: "Ready",
        status: "False",
        reason: "CreatingOrPending",
        message: "Waiting for sync.",
        lastTransitionTime: "2026-10-19T18:15:00Z",
      },
    ],
  },
];

export const mockManagedResources = [
  {
    id: "mr-1",
    name: "payments-primary-instance",
    kind: "FlexibleServer",
    group: "dbforpostgresql.azure.upbound.io",
    version: "v1beta1",
    namespace: "crossplane-system",
    environment: "prod",
    squad: "payments",
    owner: "payments",
    provider: "azure",
    region: "brazilsouth",
    resourceGroup: "rg-prod-crossplane-01",
    parentResource: "db-payments-prod",
    deletionPolicy: "Delete",
    resourceState: "OPERATIONAL",
    currentRevision: 1,
    repository: "crosswatc-org/argocd",
    gitPath: "claims/prod/db-payments.yaml",
    correlationId: "crosswatc-req-mr-01",
    synced: true,
    ready: true,
    state: "healthy",
    claimRef: { namespace: "payments", name: "payments-db-claim", kind: "PostgreSQLInstance", apiVersion: "database.example.org/v1alpha1" },
    composition: { name: "postgresql-azure-v3" },
    resourceRefsCount: 2,
    responsive: true,
    createdAt: "2026-10-17T12:55:10Z",
    updatedAt: "2026-10-19T18:00:00Z",
    conditions: [
      {
        type: "Responsive",
        status: "True",
        reason: "WatchCircuitClosed",
        message: "Resource informer active and responsive in cluster control plane.",
        lastTransitionTime: "2026-10-19T18:00:00Z",
      },
      {
        type: "Synced",
        status: "True",
        reason: "ReconcileSuccess",
        message: "Successfully reconciled Azure PostgreSQL Flexible Server.",
        lastTransitionTime: "2026-10-19T18:00:00Z",
      },
      {
        type: "Ready",
        status: "True",
        reason: "Available",
        message: "Azure Flexible Server is ready and accepting connections.",
        lastTransitionTime: "2026-10-19T18:00:00Z",
      },
    ],
  },
  {
    id: "mr-2",
    name: "payments-firewall-rule",
    kind: "FirewallRule",
    group: "dbforpostgresql.azure.upbound.io",
    version: "v1beta1",
    namespace: "crossplane-system",
    environment: "prod",
    squad: "payments",
    owner: "payments",
    provider: "azure",
    region: "brazilsouth",
    resourceGroup: "rg-prod-crossplane-01",
    parentResource: "db-payments-prod",
    deletionPolicy: "Delete",
    resourceState: "OPERATIONAL",
    currentRevision: 1,
    repository: "crosswatc-org/argocd",
    gitPath: "claims/prod/db-payments.yaml",
    correlationId: "crosswatc-req-mr-01",
    synced: true,
    ready: true,
    state: "healthy",
    claimRef: { namespace: "payments", name: "payments-db-claim", kind: "PostgreSQLInstance", apiVersion: "database.example.org/v1alpha1" },
    composition: { name: "postgresql-azure-v3" },
    resourceRefsCount: 2,
    responsive: true,
    createdAt: "2026-10-17T12:56:00Z",
    updatedAt: "2026-10-19T18:00:00Z",
    conditions: [
      {
        type: "Responsive",
        status: "True",
        reason: "WatchCircuitClosed",
        message: "Resource informer active and responsive.",
        lastTransitionTime: "2026-10-19T18:00:00Z",
      },
      {
        type: "Synced",
        status: "True",
        reason: "ReconcileSuccess",
        message: "Firewall rule reconciled.",
        lastTransitionTime: "2026-10-19T18:00:00Z",
      },
      {
        type: "Ready",
        status: "True",
        reason: "Available",
        message: "Firewall rule active.",
        lastTransitionTime: "2026-10-19T18:00:00Z",
      },
    ],
  },
  {
    id: "mr-3",
    name: "redis-node-pool",
    kind: "RedisCluster",
    group: "cache.azure.upbound.io",
    version: "v1beta1",
    namespace: "crossplane-system",
    environment: "staging",
    squad: "platform",
    owner: "platform",
    provider: "azure",
    region: "eastus2",
    resourceGroup: "rg-staging-crossplane-01",
    parentResource: "redis-session-cache",
    deletionPolicy: "Delete",
    resourceState: "PROVISIONING",
    currentRevision: 1,
    repository: "crosswatc-org/argocd",
    gitPath: "claims/staging/redis-session.yaml",
    correlationId: "crosswatc-req-mr-03",
    synced: true,
    ready: false,
    state: "progressing",
    claimRef: { namespace: "platform", name: "redis-cache-claim", kind: "RedisCluster", apiVersion: "cache.example.org/v1alpha1" },
    composition: { name: "redis-azure-v2" },
    resourceRefsCount: 1,
    responsive: true,
    createdAt: "2026-10-18T10:05:00Z",
    updatedAt: "2026-10-19T18:10:00Z",
    conditions: [
      {
        type: "Responsive",
        status: "True",
        reason: "WatchCircuitClosed",
        message: "Resource informer active.",
        lastTransitionTime: "2026-10-19T18:10:00Z",
      },
      {
        type: "Synced",
        status: "True",
        reason: "ReconcileSuccess",
        message: "Azure Redis cluster manifest applied.",
        lastTransitionTime: "2026-10-19T18:10:00Z",
      },
      {
        type: "Ready",
        status: "False",
        reason: "CreatingOrPending",
        message: "Azure Redis nodes provisioning in region eastus2.",
        lastTransitionTime: "2026-10-19T18:10:00Z",
      },
    ],
  },
];

export const mockTopology = {
  nodes: [
    {
      id: "claim-1",
      name: "payments-db-claim",
      kind: "PostgreSQLInstance",
      type: "claim",
      status: "ok",
      conditions: [{ type: "Ready", status: "True" }],
    },
    {
      id: "xr-1",
      name: "db-payments-prod",
      kind: "XPostgreSQLInstance",
      type: "composite",
      status: "ok",
      conditions: [
        {
          type: "Responsive",
          status: "True",
          reason: "WatchCircuitClosed",
          message: "Resource informer active and responsive in cluster control plane.",
        },
        {
          type: "Synced",
          status: "True",
          reason: "ReconcileSuccess",
          message: "Successfully reconciled composite resource.",
        },
        {
          type: "Ready",
          status: "True",
          reason: "Available",
          message: "Resource is available and all managed resources are ready.",
        },
      ],
    },
    {
      id: "mr-1",
      name: "payments-primary-instance",
      kind: "FlexibleServer",
      type: "managed",
      status: "ok",
      conditions: [{ type: "Ready", status: "True" }],
    },
    {
      id: "mr-2",
      name: "payments-firewall-rule",
      kind: "FirewallRule",
      type: "managed",
      status: "ok",
      conditions: [{ type: "Ready", status: "True" }],
    },
  ],
  edges: [
    { source: "claim-1", target: "xr-1" },
    { source: "xr-1", target: "mr-1" },
    { source: "xr-1", target: "mr-2" },
  ],
};

export const mockManifest = {
  object: { kind: "XPostgreSQLInstance", name: "db-payments-prod" },
  objects: [
    { kind: "XPostgreSQLInstance", name: "db-payments-prod" },
    { kind: "FlexibleServer", name: "payments-primary-instance" },
  ],
  yaml: `apiVersion: database.example.org/v1alpha1
kind: XPostgreSQLInstance
metadata:
  name: db-payments-prod
  namespace: default
spec:
  parameters:
    storageGB: 50
    engineVersion: "15"
status:
  conditions:
  - type: Synced
    status: "True"
  - type: Ready
    status: "True"
`,
};

export const mockKubeEvents = {
  items: [
    {
      type: "Normal",
      reason: "ReconcileSuccess",
      message: "Composite resource successfully synced with cloud provider",
      object: { kind: "XPostgreSQLInstance", name: "db-payments-prod" },
      count: 2,
      lastSeen: "2026-10-19T18:00:00Z",
    },
    {
      type: "Warning",
      reason: "SlowReconcile",
      message: "Provider took longer than expected to acknowledge resource update",
      object: { kind: "FlexibleServer", name: "payments-primary-instance" },
      count: 1,
      lastSeen: "2026-10-19T18:05:00Z",
    },
  ],
  errors: [],
};

export function createMockServer() {
  return http.createServer((req, res) => {
    const url = new URL(req.url ?? "/", `http://${req.headers.host ?? "localhost"}`);
    const pathname = url.pathname;

    res.setHeader("Content-Type", "application/json");
    res.setHeader("Access-Control-Allow-Origin", "*");
    res.setHeader("Access-Control-Allow-Methods", "GET, POST, OPTIONS");
    res.setHeader("Access-Control-Allow-Headers", "*");

    if (req.method === "OPTIONS") {
      res.writeHead(204);
      res.end();
      return;
    }

    // Health
    if (pathname === "/healthz" || pathname === "/readyz") {
      res.writeHead(200);
      res.end(JSON.stringify({ status: "ok" }));
      return;
    }

    // Stream SSE
    if (pathname === "/api/v1/stream") {
      res.writeHead(200, {
        "Content-Type": "text/event-stream",
        "Cache-Control": "no-cache",
        Connection: "keep-alive",
      });
      res.write("event: ping\ndata: {}\n\n");
      const interval = setInterval(() => {
        res.write("event: ping\ndata: {}\n\n");
      }, 10000);
      req.on("close", () => clearInterval(interval));
      return;
    }

    // Auth / Me
    if (pathname === "/api/v1/me") {
      res.writeHead(200);
      res.end(
        JSON.stringify({
          user: "admin",
          email: "admin@crosswatc.local",
          squads: ["sre", "platform", "payments"],
          sre: true,
          roles: ["admin", "editor"],
        })
      );
      return;
    }

    // Dashboard
    if (pathname === "/api/v1/dashboard") {
      res.writeHead(200);
      res.end(
        JSON.stringify({
          total: 3,
          synced: 2,
          ready: 1,
          unhealthy: 1,
          byKind: [
            { kind: "XPostgreSQLInstance", count: 1 },
            { kind: "XRedisCluster", count: 1 },
            { kind: "XBucket", count: 1 },
          ],
        })
      );
      return;
    }

    // Resource Kinds
    if (pathname === "/api/v1/resources/kinds") {
      res.writeHead(200);
      res.end(
        JSON.stringify({
          items: [
            { kind: "XPostgreSQLInstance", count: 12 },
            { kind: "XRedisCluster", count: 5 },
            { kind: "XBucket", count: 8 },
          ],
        })
      );
      return;
    }

    // Single Resource Sub-endpoints (topology, manifest, kube-events)
    const topologyMatch = pathname.match(/^\/api\/v1\/resources\/([^/]+)\/topology$/);
    if (topologyMatch) {
      res.writeHead(200);
      res.end(JSON.stringify(mockTopology));
      return;
    }

    const manifestMatch = pathname.match(/^\/api\/v1\/resources\/([^/]+)\/manifest$/);
    if (manifestMatch) {
      res.writeHead(200);
      res.end(JSON.stringify(mockManifest));
      return;
    }

    const kubeEventsMatch = pathname.match(/^\/api\/v1\/resources\/([^/]+)\/kube-events$/);
    if (kubeEventsMatch) {
      res.writeHead(200);
      res.end(JSON.stringify(mockKubeEvents));
      return;
    }

    const stateMatch = pathname.match(/^\/api\/v1\/resources\/([^/]+)\/state$/);
    if (stateMatch) {
      const id = stateMatch[1];
      const r = mockResources.find((resItem) => resItem.id === id || resItem.name === id)
        || mockManagedResources.find((resItem) => resItem.id === id || resItem.name === id)
        || mockResources[0];
      const isOperational = r.resourceState === "OPERATIONAL" || r.ready;
      res.writeHead(200);
      res.end(
        JSON.stringify({
          resourceState: r.resourceState,
          currentRevision: {
            rev: r.currentRevision ?? 1,
            state: isOperational ? "SUCCEEDED" : "RECONCILING",
            prUrl: `https://github.com/${r.repository}/pull/112`,
          },
          stages: [
            {
              stage: "REQUEST",
              status: "SUCCEEDED",
              actionOwner: "platform",
              message: "Solicitação validada e gerada via Backstage Scaffolder",
            },
            {
              stage: "PULL_REQUEST",
              status: "SUCCEEDED",
              actionOwner: "platform",
              message: "Pull Request #112 criado e aprovado pelos CODEOWNERS",
            },
            {
              stage: "CI_POLICIES",
              status: "SUCCEEDED",
              actionOwner: "platform",
              message: "GitHub Actions e validações OPA / Kyverno aprovados",
            },
            {
              stage: "ARGO_CD",
              status: "SUCCEEDED",
              actionOwner: "platform",
              message: "ApplicationSet sincronizado com cluster de controle (Prune=false)",
            },
            {
              stage: "CROSSPLANE",
              status: isOperational ? "SUCCEEDED" : "IN_PROGRESS",
              actionOwner: "platform",
              message: isOperational
                ? "Composition expandida e Managed Resources gerados"
                : "Reconciliando Composition corporativa",
            },
            {
              stage: "CLOUD",
              status: isOperational ? "SUCCEEDED" : "IN_PROGRESS",
              actionOwner: "platform",
              message: isOperational
                ? `Recursos físicos provisionados no provedor ${r.provider}`
                : `Aguardando provisionamento físico na nuvem ${r.provider}`,
            },
            {
              stage: "OPERATIONAL",
              status: isOperational ? "SUCCEEDED" : "PENDING",
              actionOwner: "platform",
              message: isOperational
                ? "Recurso 100% operacional; connection secret injetado"
                : "Aguardando conclusão das etapas anteriores",
            },
          ],
          drift: { detected: false },
          lastReconcileAt: r.updatedAt,
        })
      );
      return;
    }

    // Single Resource
    const singleMatch = pathname.match(/^\/api\/v1\/resources\/([^/]+)$/);
    if (singleMatch) {
      const id = singleMatch[1];
      const r = mockResources.find((resItem) => resItem.id === id || resItem.name === id)
        || mockManagedResources.find((resItem) => resItem.id === id || resItem.name === id);
      if (r) {
        res.writeHead(200);
        res.end(JSON.stringify(r));
      } else {
        // Return default first resource if unknown ID for lenient testing, or 404
        if (id === "non-existent") {
          res.writeHead(404);
          res.end(JSON.stringify({ title: "Not Found", status: 404 }));
        } else {
          res.writeHead(200);
          res.end(JSON.stringify(mockResources[0]));
        }
      }
      return;
    }

    // Resource list
    if (pathname === "/api/v1/resources") {
      const q = url.searchParams.get("q")?.toLowerCase();
      const kind = url.searchParams.get("kind");
      const synced = url.searchParams.get("synced");
      const ready = url.searchParams.get("ready");
      const responsive = url.searchParams.get("responsive");
      const composition = url.searchParams.get("composition");
      const view = url.searchParams.get("view");

      const source = view === "managed" ? mockManagedResources : mockResources;
      let filtered = [...source];
      if (q) {
        filtered = filtered.filter(
          (r) => r.name.toLowerCase().includes(q) || r.kind.toLowerCase().includes(q)
        );
      }
      if (kind) {
        filtered = filtered.filter((r) => r.kind === kind);
      }
      if (synced !== null && synced !== undefined && synced !== "") {
        filtered = filtered.filter((r) => String(r.synced) === synced);
      }
      if (ready !== null && ready !== undefined && ready !== "") {
        filtered = filtered.filter((r) => String(r.ready) === ready);
      }
      if (responsive !== null && responsive !== undefined && responsive !== "") {
        filtered = filtered.filter((r) => String(r.responsive ?? true) === responsive);
      }
      if (composition) {
        filtered = filtered.filter((r) => r.composition?.name === composition);
      }

      res.writeHead(200);
      res.end(JSON.stringify({ items: filtered, total: filtered.length }));
      return;
    }

    // Platform packages
    if (pathname === "/api/v1/platform/providers") {
      res.writeHead(200);
      res.end(
        JSON.stringify({
          items: [
            {
              kind: "Provider",
              name: "provider-azure-db",
              package: "xpkg.upbound.io/upbound/provider-azure-db:v1.2.0",
              currentRevision: "provider-azure-db-8f83b1",
              revisionActivationPolicy: "Automatic",
              healthy: true,
              installed: true,
              conditions: [
                { type: "Installed", status: "True" },
                { type: "Healthy", status: "True" },
              ],
            },
            {
              kind: "Provider",
              name: "provider-kubernetes",
              package: "xpkg.upbound.io/upbound/provider-kubernetes:v0.9.1",
              currentRevision: "provider-kubernetes-9c12a4",
              revisionActivationPolicy: "Automatic",
              healthy: false,
              installed: true,
              conditions: [
                { type: "Installed", status: "True" },
                { type: "Healthy", status: "False", reason: "CannotConnect", message: "Failed to connect to cluster API" },
              ],
            },
          ],
          total: 2,
        })
      );
      return;
    }

    if (pathname === "/api/v1/platform/functions") {
      res.writeHead(200);
      res.end(
        JSON.stringify({
          items: [
            {
              kind: "Function",
              name: "function-patch-and-transform",
              package: "xpkg.upbound.io/crossplane-contrib/function-patch-and-transform:v0.4.0",
              currentRevision: "function-patch-and-transform-b19c2",
              revisionActivationPolicy: "Automatic",
              healthy: true,
              installed: true,
              conditions: [
                { type: "Installed", status: "True" },
                { type: "Healthy", status: "True" },
              ],
            },
            {
              kind: "Function",
              name: "function-auto-ready",
              package: "xpkg.upbound.io/crossplane-contrib/function-auto-ready:v0.2.1",
              currentRevision: "function-auto-ready-d41a7",
              revisionActivationPolicy: "Automatic",
              healthy: true,
              installed: true,
              conditions: [
                { type: "Installed", status: "True" },
                { type: "Healthy", status: "True" },
              ],
            },
          ],
          total: 2,
        })
      );
      return;
    }

    if (pathname === "/api/v1/platform/configurations") {
      res.writeHead(200);
      res.end(
        JSON.stringify({
          items: [
            {
              kind: "Configuration",
              name: "configuration-platform-ref",
              package: "xpkg.upbound.io/platform/ref-arch:v0.1.0",
              currentRevision: "configuration-platform-ref-a1",
              revisionActivationPolicy: "Automatic",
              healthy: true,
              installed: true,
              conditions: [
                { type: "Installed", status: "True" },
                { type: "Healthy", status: "True" },
              ],
            },
          ],
          total: 1,
        })
      );
      return;
    }

    if (pathname === "/api/v1/platform/xrds") {
      res.writeHead(200);
      res.end(
        JSON.stringify({
          items: [
            {
              name: "xpostgresqlinstances.database.example.org",
              group: "database.example.org",
              kind: "XPostgreSQLInstance",
              claimKind: "PostgreSQLInstance",
              scope: "Cluster",
              version: "v1alpha1",
              established: true,
              offered: true,
            },
          ],
          total: 1,
        })
      );
      return;
    }

    if (pathname === "/api/v1/platform/compositions") {
      res.writeHead(200);
      res.end(
        JSON.stringify({
          items: [
            {
              name: "postgresql-azure-v3",
              compositeType: "XPostgreSQLInstance",
              mode: "Pipeline",
              steps: 2,
              functions: ["function-patch-and-transform", "function-auto-ready"],
              latestRevision: "postgresql-azure-v3-rev2",
            },
          ],
          total: 1,
        })
      );
      return;
    }

    if (pathname === "/api/v1/platform/mrds") {
      res.writeHead(200);
      res.end(
        JSON.stringify({
          items: [
            {
              name: "flexibleservers.azure",
              group: "azure.crossplane.io",
              kind: "FlexibleServer",
              scope: "Cluster",
              state: "Active",
              active: true,
              established: true,
              conditions: [{ type: "Established", status: "True" }],
            },
          ],
          available: true,
        })
      );
      return;
    }

    if (pathname === "/api/v1/platform/mraps") {
      res.writeHead(200);
      res.end(
        JSON.stringify({
          items: [
            {
              name: "activate-all-azure",
              activate: ["flexibleservers.azure"],
              conditions: [{ type: "Applied", status: "True" }],
            },
          ],
          available: true,
        })
      );
      return;
    }

    // Default 404
    res.writeHead(404);
    res.end(JSON.stringify({ title: "Not Found", status: 404 }));
  });
}

// Start directly if run as a standalone script
if (process.argv[1]?.endsWith("mock-backend.ts") || process.argv[1]?.endsWith("mock-backend.js")) {
  const port = Number(process.env.PORT || 8088);
  const server = createMockServer();
  server.listen(port, "127.0.0.1", () => {
    console.log(`Mock backend listening on http://127.0.0.1:${port}`);
  });
}
