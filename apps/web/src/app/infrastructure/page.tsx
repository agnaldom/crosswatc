"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { Suspense, useState } from "react";
import { Lifecycle } from "@/components/Lifecycle";
import { KubeEvents } from "@/components/KubeEvents";
import { Empty, ErrorState, Loading } from "@/components/States";
import { TopologyView } from "@/components/TopologyView";
import { WatchButton } from "@/components/WatchButton";
import { YamlViewer } from "@/components/YamlViewer";
import { api, type ResourceFilters } from "@/lib/api";
import type { ResourceState, TrackedResource } from "@/lib/types";

export default function InfrastructurePage() {
  return (
    <Suspense fallback={<Loading />}>
      <InfrastructureContent />
    </Suspense>
  );
}

function InfrastructureContent() {
  const searchParams = useSearchParams();
  const kind = searchParams.get("kind") ?? undefined;
  const view = searchParams.get("view") ?? undefined;

  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [isDetailCollapsed, setIsDetailCollapsed] = useState(false);
  const [detailTab, setDetailTab] = useState<"Overview" | "Lifecycle" | "YAML" | "Status" | "Relations" | "Events">("Relations");
  const [kubeFilter, setKubeFilter] = useState("");

  const filters: ResourceFilters = {
    kind,
    q: searchQuery || undefined,
    status: statusFilter && !["synced", "not_synced", "ready", "not_ready"].includes(statusFilter)
      ? (statusFilter as ResourceState)
      : undefined,
    synced: statusFilter === "synced" ? "true" : statusFilter === "not_synced" ? "false" : undefined,
    ready: statusFilter === "ready" ? "true" : statusFilter === "not_ready" ? "false" : undefined,
  };

  const listQuery = useQuery({
    queryKey: ["resources", kind, filters],
    queryFn: () => api.listResources({ ...filters, limit: 50 }),
  });

  const items = listQuery.data?.items ?? [];
  const selectedResource = items.find((r) => r.id === selectedId) ?? items[0] ?? null;

  const pageTitle = kind || (view === "claims" ? "Claims" : "Composite Resources");

  return (
    <div className="mx-auto max-w-7xl space-y-6">
      {/* Page Title */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-100">
          {pageTitle}
        </h1>
      </div>

      {/* Filter Bar */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex flex-wrap items-center gap-3">
          {/* Search Box */}
          <div className="relative">
            <svg
              viewBox="0 0 20 20"
              fill="currentColor"
              className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400"
            >
              <path
                fillRule="evenodd"
                d="M9 3.5a5.5 5.5 0 100 11 5.5 5.5 0 000-11zM2 9a7 7 0 1112.452 4.391l3.328 3.329a.75.75 0 11-1.06 1.06l-3.329-3.328A7 7 0 012 9z"
                clipRule="evenodd"
              />
            </svg>
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search..."
              aria-label="Search resources"
              className="h-9 w-64 rounded-lg border border-zinc-200 bg-white pl-9 pr-3 text-xs text-zinc-900 placeholder:text-zinc-400 focus:border-cyan-500 focus:outline-none dark:border-zinc-800 dark:bg-zinc-900/60 dark:text-zinc-100"
            />
          </div>

          {/* Status Dropdown */}
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            aria-label="Filter by Status"
            className="h-9 rounded-lg border border-zinc-200 bg-white px-3 text-xs text-zinc-700 focus:border-cyan-500 focus:outline-none dark:border-zinc-800 dark:bg-zinc-900/60 dark:text-zinc-300"
          >
            <option value="">All Statuses</option>
            <option value="synced">Synced</option>
            <option value="not_synced">Not Synced</option>
            <option value="ready">Ready</option>
            <option value="not_ready">Not Ready</option>
            <option value="OPERATIONAL">Operational</option>
            <option value="DEGRADED">Degraded</option>
            <option value="FAILED">Failed</option>
          </select>
        </div>

        {/* Results Counter */}
        <div className="text-xs font-medium text-zinc-500 dark:text-zinc-400">
          {items.length} {items.length === 1 ? "result" : "results"}
        </div>
      </div>

      {/* Main Table */}
      <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white shadow-sm dark:border-zinc-800 dark:bg-[#161b22]">
        {listQuery.isPending && <Loading />}
        {listQuery.isError && <ErrorState error={listQuery.error} />}
        {!listQuery.isPending && items.length === 0 && (
          <Empty>Nenhum recurso encontrado.</Empty>
        )}

        {items.length > 0 && (
          <table className="w-full text-left text-xs">
            <thead className="border-b border-zinc-200 bg-zinc-50/75 text-[11px] font-bold uppercase tracking-wider text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/50">
              <tr>
                <th className="px-5 py-3 font-semibold">Name</th>
                <th className="px-5 py-3 font-semibold">Kind</th>
                <th className="px-5 py-3 font-semibold">Status</th>
                <th className="px-5 py-3 font-semibold">Created</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-100 dark:divide-zinc-800/80">
              {items.map((r) => {
                const isSelected = selectedResource?.id === r.id;
                return (
                  <tr
                    key={r.id}
                    onClick={() => setSelectedId(r.id)}
                    className={`cursor-pointer transition ${
                      isSelected
                        ? "bg-zinc-100/90 font-medium dark:bg-zinc-800/60"
                        : "hover:bg-zinc-50/80 dark:hover:bg-zinc-800/30"
                    }`}
                  >
                    <td className="whitespace-nowrap px-5 py-3 text-zinc-900 dark:text-zinc-100">
                      <div className="flex items-center gap-2">
                        <WatchButton resource={r} />
                        <span className="truncate">{r.name}</span>
                      </div>
                    </td>
                    <td className="px-5 py-3 text-zinc-600 dark:text-zinc-300">{r.kind}</td>
                    <td className="px-5 py-3">
                      <div className="flex flex-wrap items-center gap-1.5">
                        {/* Synced status pill */}
                        <span
                          className={`rounded px-2 py-0.5 text-[10px] font-bold ${
                            r.synced
                              ? "bg-emerald-100 text-emerald-800 dark:bg-emerald-950/80 dark:text-emerald-400 dark:border dark:border-emerald-800/50"
                              : "bg-rose-100 text-rose-800 dark:bg-rose-950/80 dark:text-rose-400 dark:border dark:border-rose-800/50"
                          }`}
                        >
                          {r.synced ? "Synced" : "Not Synced"}
                        </span>

                        {/* Ready status pill */}
                        <span
                          className={`rounded px-2 py-0.5 text-[10px] font-bold ${
                            r.ready
                              ? "bg-emerald-100 text-emerald-800 dark:bg-emerald-950/80 dark:text-emerald-400 dark:border dark:border-emerald-800/50"
                              : "bg-rose-100 text-rose-800 dark:bg-rose-950/80 dark:text-rose-400 dark:border dark:border-rose-800/50"
                          }`}
                        >
                          {r.ready ? "Ready" : "Not Ready"}
                        </span>

                        {/* Responsive pill */}
                        <span className="rounded bg-emerald-900/60 px-2 py-0.5 text-[10px] font-bold text-emerald-300 border border-emerald-700/50">
                          Responsive
                        </span>
                      </div>
                    </td>
                    <td className="whitespace-nowrap px-5 py-3 text-zinc-500">
                      {r.createdAt
                        ? new Date(r.createdAt).toLocaleString("en-US", {
                            month: "numeric",
                            day: "numeric",
                            year: "numeric",
                            hour: "numeric",
                            minute: "numeric",
                            second: "numeric",
                            hour12: true,
                          })
                        : "—"}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>

      {/* Selected Resource Detail Pane (Split View below table) */}
      {selectedResource && (
        <div
          role="region"
          aria-label="Detalhes do recurso selecionado"
          className="overflow-hidden rounded-xl border border-zinc-200 bg-white shadow-sm dark:border-zinc-800 dark:bg-[#161b22]"
        >
          {/* Header of Detail Pane */}
          <div className="flex items-center justify-between border-b border-zinc-200 px-6 py-4 dark:border-zinc-800">
            <div>
              <div className="flex items-center gap-2">
                <span className="text-base font-bold text-zinc-900 dark:text-zinc-100">
                  {selectedResource.kind}
                </span>
                <span className="font-mono text-xs text-zinc-400">
                  {selectedResource.claimRef?.apiVersion || "example.org/v1alpha1"}
                </span>
              </div>
              <div className="text-xs text-zinc-500">{selectedResource.name}</div>
            </div>

            <div className="flex items-center gap-3">
              <WatchButton resource={selectedResource} />
              <button
                type="button"
                onClick={() => setIsDetailCollapsed((c) => !c)}
                title={isDetailCollapsed ? "Expandir detalhes" : "Recolher detalhes"}
                className="rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700 dark:hover:bg-zinc-800"
              >
                <svg viewBox="0 0 20 20" fill="currentColor" className="h-4 w-4">
                  <path fillRule="evenodd" d="M4 10a.75.75 0 01.75-.75h10.5a.75.75 0 010 1.5H4.75A.75.75 0 014 10z" clipRule="evenodd" />
                </svg>
              </button>
            </div>
          </div>

          {!isDetailCollapsed && (
            <div>
              {/* Tab Navigation */}
              <div className="flex border-b border-zinc-200 px-6 dark:border-zinc-800">
                {(["Overview", "Lifecycle", "YAML", "Status", "Relations", "Events"] as const).map((tab) => {
                  const isActive = detailTab === tab;
                  return (
                    <button
                      key={tab}
                      type="button"
                      onClick={() => setDetailTab(tab)}
                      className={`-mb-px border-b-2 px-4 py-2.5 text-xs font-semibold transition ${
                        isActive
                          ? "border-cyan-500 text-cyan-600 dark:border-cyan-400 dark:text-cyan-400"
                          : "border-transparent text-zinc-500 hover:border-zinc-300 hover:text-zinc-800 dark:hover:text-zinc-200"
                      }`}
                    >
                      {tab}
                    </button>
                  );
                })}
              </div>

              {/* Tab Contents */}
              <div className="p-6">
                {/* Lifecycle Tab */}
                {detailTab === "Lifecycle" && (
                  <LifecycleTabContent resource={selectedResource} />
                )}

                {/* Relations Tab */}
                {detailTab === "Relations" && (
                  <RelationsTabContent resourceId={selectedResource.id} />
                )}

                {/* YAML Tab */}
                {detailTab === "YAML" && (
                  <YamlTabContent resourceId={selectedResource.id} />
                )}

                {/* Status Tab */}
                {detailTab === "Status" && (
                  <StatusTabContent resource={selectedResource} />
                )}

                {/* Events Tab */}
                {detailTab === "Events" && (
                  <KubeEvents id={selectedResource.id} filter={kubeFilter} onFilter={setKubeFilter} />
                )}

                {/* Overview Tab */}
                {detailTab === "Overview" && (
                  <OverviewTabContent
                    resource={selectedResource}
                    onOpenLifecycle={() => setDetailTab("Lifecycle")}
                  />
                )}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function RelationsTabContent({ resourceId }: { resourceId: string }) {
  const q = useQuery({
    queryKey: ["topology", resourceId],
    queryFn: () => api.topology(resourceId),
  });

  if (q.isPending) return <Loading />;
  if (q.isError) return <ErrorState error={q.error} />;
  return <TopologyView topology={q.data} />;
}

function YamlTabContent({ resourceId }: { resourceId: string }) {
  const q = useQuery({
    queryKey: ["manifest", resourceId],
    queryFn: () => api.manifest(resourceId),
  });

  if (q.isPending) return <Loading />;
  if (q.isError) return <ErrorState error={q.error} />;
  return <YamlViewer yaml={q.data.yaml} />;
}

function StatusTabContent({ resource }: { resource: TrackedResource }) {
  const topologyQuery = useQuery({
    queryKey: ["topology", resource.id],
    queryFn: () => api.topology(resource.id),
  });

  const centralNode = topologyQuery.data?.nodes.find((n) => n.id.includes(resource.name) || n.name === resource.name || n.type === "composite");
  const conditions = centralNode?.conditions ?? [
    {
      type: "Responsive",
      status: "True",
      reason: "WatchCircuitClosed",
      message: "Resource informer active and responsive in cluster control plane.",
      lastTransitionTime: resource.updatedAt,
    },
    {
      type: "Synced",
      status: resource.synced ? "True" : resource.synced === false ? "False" : "Unknown",
      reason: resource.synced ? "ReconcileSuccess" : "ReconcilePending",
      message: resource.synced === false ? `[cannot compose resources: ${resource.name} pending synchronization]` : undefined,
      lastTransitionTime: resource.updatedAt,
    },
    {
      type: "Ready",
      status: resource.ready ? "True" : resource.ready === false ? "False" : "Unknown",
      reason: resource.ready ? "Available" : "CreatingOrPending",
      message: resource.ready === false ? "Managed resources pending deployment in target cloud provider." : undefined,
      lastTransitionTime: resource.updatedAt,
    },
  ];

  return (
    <div className="grid gap-4 sm:grid-cols-2 md:grid-cols-3">
      {conditions.map((c, i) => {
        const isTrue = c.status?.toLowerCase() === "true" || c.status === "True";
        return (
          <div
            key={i}
            className={`rounded-xl border p-4 text-left transition ${
              isTrue
                ? "border-emerald-200/80 bg-emerald-50/30 dark:border-emerald-900/40 dark:bg-emerald-950/20"
                : "border-rose-200/80 bg-rose-50/30 dark:border-rose-900/40 dark:bg-rose-950/20"
            }`}
          >
            <div className="flex items-center justify-between">
              <span className="text-xs font-bold text-zinc-900 dark:text-zinc-100">{c.type}</span>
              <span
                className={`rounded px-2 py-0.5 text-[10px] font-bold ${
                  isTrue
                    ? "bg-emerald-600 text-white dark:bg-emerald-500"
                    : "bg-rose-600 text-white dark:bg-rose-500"
                }`}
              >
                {c.status}
              </span>
            </div>
            {c.reason && (
              <div className="mt-2 text-xs text-zinc-600 dark:text-zinc-400">
                <span className="font-medium text-zinc-500">Reason:</span> {c.reason}
              </div>
            )}
            {c.message && (
              <div className="mt-2 max-h-28 overflow-y-auto rounded bg-zinc-100 p-2 font-mono text-[10px] text-zinc-700 dark:bg-zinc-900/80 dark:text-zinc-300">
                {c.message}
              </div>
            )}
            {c.lastTransitionTime && (
              <div className="mt-2 text-[10px] text-zinc-400">
                Last Transition: {new Date(c.lastTransitionTime).toLocaleString("pt-BR")}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

function LifecycleTabContent({ resource }: { resource: TrackedResource }) {
  const stateQuery = useQuery({
    queryKey: ["state", resource.id],
    queryFn: () => api.state(resource.id),
  });

  if (stateQuery.isPending) return <Loading />;
  if (stateQuery.isError) return <ErrorState error={stateQuery.error} />;

  return <Lifecycle stages={stateQuery.data?.stages} resource={resource} />;
}

function OverviewTabContent({
  resource: r,
  onOpenLifecycle,
}: {
  resource: TrackedResource;
  onOpenLifecycle?: () => void;
}) {
  return (
    <div className="grid gap-6 md:grid-cols-2">
      <div className="rounded-xl border border-zinc-200 bg-zinc-50/50 p-4 dark:border-zinc-800 dark:bg-zinc-900/30">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-zinc-500">Metadados</h3>
        <dl className="mt-3 grid grid-cols-[7rem_1fr] gap-y-2 text-xs">
          <dt className="text-zinc-400">Owner</dt>
          <dd className="font-medium text-zinc-900 dark:text-zinc-100">{r.owner}</dd>
          <dt className="text-zinc-400">Region</dt>
          <dd>{r.region || "—"}</dd>
          <dt className="text-zinc-400">Claim</dt>
          <dd className="font-mono text-[11px]">{r.claimRef?.namespace}/{r.claimRef?.name}</dd>
          <dt className="text-zinc-400">Composition</dt>
          <dd className="font-mono text-[11px]">{r.composition?.name || "—"}</dd>
        </dl>
      </div>

      <div className="rounded-xl border border-zinc-200 bg-zinc-50/50 p-4 dark:border-zinc-800 dark:bg-zinc-900/30">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-zinc-500">GitOps & Rastreabilidade</h3>
        <dl className="mt-3 grid grid-cols-[7rem_1fr] gap-y-2 text-xs">
          <dt className="text-zinc-400">Repository</dt>
          <dd className="font-mono text-[11px] truncate">{r.repository || "—"}</dd>
          <dt className="text-zinc-400">Git Path</dt>
          <dd className="break-all font-mono text-[11px]">{r.gitPath || "—"}</dd>
          <dt className="text-zinc-400">Correlation ID</dt>
          <dd className="break-all font-mono text-[10px] text-zinc-500">{r.correlationId}</dd>
        </dl>
        <div className="mt-4 flex flex-wrap gap-2">
          {onOpenLifecycle && (
            <button
              type="button"
              onClick={onOpenLifecycle}
              className="rounded-md bg-cyan-600 px-3 py-1.5 text-xs font-medium text-white shadow-sm transition hover:bg-cyan-500"
            >
              Ver Lifecycle no Painel ↓
            </button>
          )}
          <Link
            href={`/infrastructure/${r.id}?tab=Lifecycle`}
            className="rounded-md border border-zinc-300 bg-white px-3 py-1.5 text-xs font-medium text-zinc-700 transition hover:bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-200 dark:hover:bg-zinc-700"
          >
            Abrir Lifecycle Completo ↗
          </Link>
        </div>
      </div>
    </div>
  );
}
