"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { Suspense, useMemo, useState } from "react";
import { Lifecycle } from "@/components/Lifecycle";
import { KubeEvents } from "@/components/KubeEvents";
import { ErrorState, Loading } from "@/components/States";
import { TopologyView } from "@/components/TopologyView";
import { WatchButton } from "@/components/WatchButton";
import { YamlViewer } from "@/components/YamlViewer";
import { api, type ResourceFilters } from "@/lib/api";
import type { TrackedResource } from "@/lib/types";

export default function InfrastructurePage() {
  return (
    <Suspense fallback={<Loading />}>
      <InfrastructureContent />
    </Suspense>
  );
}

function InfrastructureContent() {
  const searchParams = useSearchParams();
  const kindParam = searchParams.get("kind") ?? undefined;
  const rawView = searchParams.get("view");
  const view: "claims" | "composites" | "managed" =
    rawView === "claims" ? "claims" : rawView === "managed" ? "managed" : "composites";

  const [searchQuery, setSearchQuery] = useState("");
  const [kindFilter, setKindFilter] = useState(kindParam || "");
  const [syncedFilter, setSyncedFilter] = useState("");
  const [readyFilter, setReadyFilter] = useState("");
  const [responsiveFilter, setResponsiveFilter] = useState("");
  const [compositionFilter, setCompositionFilter] = useState("");

  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [isDetailCollapsed, setIsDetailCollapsed] = useState(false);
  const [detailTab, setDetailTab] = useState<"Overview" | "Lifecycle" | "YAML" | "Status" | "Relations" | "Events">("Relations");
  const [kubeFilter, setKubeFilter] = useState("");

  const kindsQuery = useQuery({
    queryKey: ["resource-kinds"],
    queryFn: api.resourceKinds,
    staleTime: 60_000,
  });

  const compositionsQuery = useQuery({
    queryKey: ["platform-compositions"],
    queryFn: () => api.platformCompositions().catch(() => ({ items: [], nextCursor: null })),
    staleTime: 60_000,
  });

  const filters: ResourceFilters = {
    view,
    kind: kindFilter || kindParam || undefined,
    q: searchQuery || undefined,
    synced: syncedFilter || undefined,
    ready: readyFilter || undefined,
    responsive: responsiveFilter || undefined,
    composition: compositionFilter || undefined,
  };

  const listQuery = useQuery({
    queryKey: ["resources", view, kindParam, filters],
    queryFn: () => api.listResources({ ...filters, limit: 50 }),
  });

  const items = useMemo(() => listQuery.data?.items ?? [], [listQuery.data?.items]);
  const selectedResource = items.find((r) => r.id === selectedId) ?? items[0] ?? null;

  // Derive unique options
  const availableKinds = useMemo(() => {
    const fromApi = kindsQuery.data?.items?.map((k) => k.kind) || [];
    const fromItems = Array.from(new Set(items.map((r) => r.kind)));
    return Array.from(new Set([...fromApi, ...fromItems])).filter(Boolean);
  }, [kindsQuery.data, items]);

  const availableCompositions = useMemo(() => {
    const fromApi = compositionsQuery.data?.items?.map((c) => c.name) || [];
    const fromItems = Array.from(
      new Set(items.map((r) => r.composition?.name).filter(Boolean) as string[])
    );
    return Array.from(new Set([...fromApi, ...fromItems])).filter(Boolean);
  }, [compositionsQuery.data, items]);

  const pageTitle =
    view === "claims"
      ? "Claims"
      : view === "managed"
      ? "Managed Resources"
      : kindParam || "Composite Resources";

  return (
    <div className="mx-auto max-w-7xl space-y-6">
      {/* Header and View Navigation */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-100">
            {pageTitle}
          </h1>
        </div>

        {/* View Switcher Pills */}
        <div className="flex items-center gap-1 rounded-xl border border-zinc-200 bg-zinc-100/80 p-1 dark:border-zinc-800 dark:bg-zinc-900/80">
          <Link
            href="/infrastructure?view=composites"
            className={`flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-semibold transition ${
              view === "composites" && !kindParam
                ? "bg-white text-zinc-900 shadow-sm dark:bg-zinc-800 dark:text-white"
                : "text-zinc-500 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white"
            }`}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-3.5 w-3.5">
              <rect x="2" y="7" width="20" height="14" rx="2" ry="2" />
              <path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16" />
            </svg>
            <span>Composites (XRs)</span>
          </Link>

          <Link
            href="/infrastructure?view=claims"
            className={`flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-semibold transition ${
              view === "claims"
                ? "bg-white text-zinc-900 shadow-sm dark:bg-zinc-800 dark:text-white"
                : "text-zinc-500 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white"
            }`}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-3.5 w-3.5">
              <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2" />
              <rect x="8" y="2" width="8" height="4" rx="1" ry="1" />
            </svg>
            <span>Claims (XRCs)</span>
          </Link>

          <Link
            href="/infrastructure?view=managed"
            className={`flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-semibold transition ${
              view === "managed"
                ? "bg-white text-zinc-900 shadow-sm dark:bg-zinc-800 dark:text-white"
                : "text-zinc-500 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white"
            }`}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-3.5 w-3.5">
              <ellipse cx="12" cy="5" rx="9" ry="3" />
              <path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3" />
              <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5" />
            </svg>
            <span>Managed (MRs)</span>
          </Link>
        </div>
      </div>

      {/* Filter Bar with Search and Dropdowns */}
      <div className="space-y-2">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex flex-wrap items-center gap-2.5">
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
                className="h-9 w-60 rounded-lg border border-zinc-200 bg-white pl-9 pr-3 text-xs text-zinc-900 placeholder:text-zinc-400 focus:border-cyan-500 focus:outline-none dark:border-zinc-800 dark:bg-zinc-900/60 dark:text-zinc-100"
              />
            </div>

            {/* Dropdown: All Kinds */}
            <select
              value={kindFilter}
              onChange={(e) => setKindFilter(e.target.value)}
              aria-label="Filter by Kind"
              className="h-9 rounded-lg border border-zinc-200 bg-white px-3 text-xs font-medium text-zinc-700 focus:border-cyan-500 focus:outline-none dark:border-zinc-800 dark:bg-zinc-900/60 dark:text-zinc-300"
            >
              <option value="">All Kinds</option>
              {availableKinds.map((k) => (
                <option key={k} value={k}>
                  {k}
                </option>
              ))}
            </select>

            {/* Dropdown: All Synced */}
            <select
              value={syncedFilter}
              onChange={(e) => setSyncedFilter(e.target.value)}
              aria-label="Filter by Synced"
              className="h-9 rounded-lg border border-zinc-200 bg-white px-3 text-xs font-medium text-zinc-700 focus:border-cyan-500 focus:outline-none dark:border-zinc-800 dark:bg-zinc-900/60 dark:text-zinc-300"
            >
              <option value="">All Synced</option>
              <option value="true">Synced</option>
              <option value="false">Not Synced</option>
            </select>

            {/* Dropdown: All Ready */}
            <select
              value={readyFilter}
              onChange={(e) => setReadyFilter(e.target.value)}
              aria-label="Filter by Ready"
              className="h-9 rounded-lg border border-zinc-200 bg-white px-3 text-xs font-medium text-zinc-700 focus:border-cyan-500 focus:outline-none dark:border-zinc-800 dark:bg-zinc-900/60 dark:text-zinc-300"
            >
              <option value="">All Ready</option>
              <option value="true">Ready</option>
              <option value="false">Not Ready</option>
            </select>

            {/* Dropdown: All Responsive */}
            <select
              value={responsiveFilter}
              onChange={(e) => setResponsiveFilter(e.target.value)}
              aria-label="Filter by Responsive"
              className="h-9 rounded-lg border border-zinc-200 bg-white px-3 text-xs font-medium text-zinc-700 focus:border-cyan-500 focus:outline-none dark:border-zinc-800 dark:bg-zinc-900/60 dark:text-zinc-300"
            >
              <option value="">All Responsive</option>
              <option value="true">Responsive</option>
              <option value="false">Not Responsive</option>
            </select>

            {/* Dropdown: All Compositions */}
            <select
              value={compositionFilter}
              onChange={(e) => setCompositionFilter(e.target.value)}
              aria-label="Filter by Composition"
              className="h-9 rounded-lg border border-zinc-200 bg-white px-3 text-xs font-medium text-zinc-700 focus:border-cyan-500 focus:outline-none dark:border-zinc-800 dark:bg-zinc-900/60 dark:text-zinc-300"
            >
              <option value="">All Compositions</option>
              {availableCompositions.map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </select>
          </div>

          {/* Results Counter aligned to right */}
          <div className="text-xs font-medium text-zinc-500 dark:text-zinc-400">
            {items.length} {items.length === 1 ? "result" : "results"}
          </div>
        </div>
      </div>

      {/* Main Table Card */}
      <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white shadow-sm dark:border-zinc-800 dark:bg-[#161b22]">
        {listQuery.isPending && <Loading />}
        {listQuery.isError && <ErrorState error={listQuery.error} />}

        {!listQuery.isPending && items.length === 0 && (
          <div className="py-16 text-center text-sm font-medium text-zinc-400 dark:text-zinc-500">
            No data found
          </div>
        )}

        {items.length > 0 && (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="border-b border-zinc-200 bg-zinc-50/75 text-[11px] font-bold uppercase tracking-wider text-zinc-600 dark:border-zinc-800 dark:bg-zinc-900/50 dark:text-zinc-400">
                <tr>
                  <th className="px-5 py-3 font-bold">NAME</th>
                  <th className="px-5 py-3 font-bold">KIND</th>

                  {view === "claims" ? (
                    <>
                      <th className="px-5 py-3 font-bold">NAMESPACE</th>
                      <th className="px-5 py-3 font-bold">OWNER / SQUAD</th>
                      <th className="px-5 py-3 font-bold">COMPOSITE TARGET</th>
                    </>
                  ) : view === "managed" ? (
                    <>
                      <th className="px-5 py-3 font-bold">PROVIDER</th>
                      <th className="px-5 py-3 font-bold">PARENT COMPOSITE</th>
                      <th className="px-5 py-3 font-bold">DELETION POLICY</th>
                    </>
                  ) : (
                    <>
                      <th className="px-5 py-3 font-bold">SYNCED</th>
                      <th className="px-5 py-3 font-bold">READY</th>
                      <th className="px-5 py-3 font-bold">RESPONSIVE</th>
                      <th className="px-5 py-3 font-bold">COMPOSITION</th>
                      <th className="px-5 py-3 font-bold">CLAIM REF</th>
                      <th className="px-5 py-3 font-bold">RESOURCE REFS</th>
                    </>
                  )}

                  {view !== "composites" && (
                    <>
                      <th className="px-5 py-3 font-bold">SYNCED</th>
                      <th className="px-5 py-3 font-bold">READY</th>
                      <th className="px-5 py-3 font-bold">RESPONSIVE</th>
                    </>
                  )}

                  <th className="px-5 py-3 font-bold">CREATED</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-100 dark:divide-zinc-800/80">
                {items.map((r) => {
                  const isSelected = selectedResource?.id === r.id;
                  const displayName =
                    view === "claims" ? r.claimRef?.name || r.name : r.name;
                  const displayKind =
                    view === "claims" ? r.claimRef?.kind || r.kind : r.kind;

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
                      {/* NAME */}
                      <td className="whitespace-nowrap px-5 py-3 text-zinc-900 dark:text-zinc-100">
                        <div className="flex items-center gap-2">
                          <WatchButton resource={r} />
                          <span className="truncate font-semibold">{displayName}</span>
                        </div>
                      </td>

                      {/* KIND */}
                      <td className="px-5 py-3 font-mono text-zinc-700 dark:text-zinc-300">
                        {displayKind}
                      </td>

                      {/* SPECIFIC COLUMNS */}
                      {view === "claims" && (
                        <>
                          <td className="px-5 py-3">
                            <span className="rounded bg-zinc-100 px-2 py-0.5 font-mono text-[11px] text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300">
                              {r.claimRef?.namespace || "default"}
                            </span>
                          </td>
                          <td className="px-5 py-3 text-zinc-600 dark:text-zinc-300">
                            {r.owner || "—"}
                          </td>
                          <td className="px-5 py-3 font-mono text-[11px] text-cyan-600 dark:text-cyan-400">
                            {r.name}
                          </td>
                        </>
                      )}

                      {view === "managed" && (
                        <>
                          <td className="px-5 py-3">
                            <span className="inline-flex items-center rounded-md border border-sky-800/40 bg-sky-950/40 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-sky-400">
                              {r.provider || "cloud"}
                            </span>
                          </td>
                          <td className="px-5 py-3 font-mono text-[11px] text-zinc-500 dark:text-zinc-400">
                            {r.parentResource || r.claimRef?.name || r.name}
                          </td>
                          <td className="px-5 py-3">
                            <span
                              className={`rounded px-2 py-0.5 text-[10px] font-semibold ${
                                r.deletionPolicy === "Orphan"
                                  ? "bg-amber-100 text-amber-800 dark:bg-amber-950/60 dark:text-amber-400"
                                  : "bg-zinc-100 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300"
                              }`}
                            >
                              {r.deletionPolicy || "Delete"}
                            </span>
                          </td>
                        </>
                      )}

                      {view === "composites" && (
                        <>
                          {/* SYNCED */}
                          <td className="px-5 py-3">
                            <span
                              className={`rounded px-2 py-0.5 text-[10px] font-bold ${
                                r.synced
                                  ? "bg-emerald-100 text-emerald-800 dark:bg-emerald-950/80 dark:text-emerald-400 dark:border dark:border-emerald-800/50"
                                  : "bg-rose-100 text-rose-800 dark:bg-rose-950/80 dark:text-rose-400 dark:border dark:border-rose-800/50"
                              }`}
                            >
                              {r.synced ? "Synced" : "Not Synced"}
                            </span>
                          </td>

                          {/* READY */}
                          <td className="px-5 py-3">
                            <span
                              className={`rounded px-2 py-0.5 text-[10px] font-bold ${
                                r.ready
                                  ? "bg-emerald-100 text-emerald-800 dark:bg-emerald-950/80 dark:text-emerald-400 dark:border dark:border-emerald-800/50"
                                  : "bg-rose-100 text-rose-800 dark:bg-rose-950/80 dark:text-rose-400 dark:border dark:border-rose-800/50"
                              }`}
                            >
                              {r.ready ? "Ready" : "Not Ready"}
                            </span>
                          </td>

                          {/* RESPONSIVE */}
                          <td className="px-5 py-3">
                            <span className="rounded border border-emerald-700/50 bg-emerald-900/60 px-2 py-0.5 text-[10px] font-bold text-emerald-300">
                              Responsive
                            </span>
                          </td>

                          {/* COMPOSITION */}
                          <td className="px-5 py-3 font-mono text-[11px] text-zinc-600 dark:text-zinc-400">
                            {r.composition?.name || "—"}
                          </td>

                          {/* CLAIM REF */}
                          <td className="px-5 py-3 font-mono text-[11px] text-zinc-600 dark:text-zinc-400">
                            {r.claimRef
                              ? `${r.claimRef.namespace} / ${r.claimRef.name}`
                              : "—"}
                          </td>

                          {/* RESOURCE REFS */}
                          <td className="px-5 py-3">
                            <span className="rounded-md border border-zinc-200 bg-zinc-50 px-2 py-0.5 text-[11px] font-medium text-zinc-700 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-300">
                              {r.resourceRefsCount || 2} resources
                            </span>
                          </td>
                        </>
                      )}

                      {/* NON-COMPOSITES STATUS COLUMNS */}
                      {view !== "composites" && (
                        <>
                          <td className="px-5 py-3">
                            <span
                              className={`rounded px-2 py-0.5 text-[10px] font-bold ${
                                r.synced
                                  ? "bg-emerald-100 text-emerald-800 dark:bg-emerald-950/80 dark:text-emerald-400"
                                  : "bg-rose-100 text-rose-800 dark:bg-rose-950/80 dark:text-rose-400"
                              }`}
                            >
                              {r.synced ? "Synced" : "Not Synced"}
                            </span>
                          </td>
                          <td className="px-5 py-3">
                            <span
                              className={`rounded px-2 py-0.5 text-[10px] font-bold ${
                                r.ready
                                  ? "bg-emerald-100 text-emerald-800 dark:bg-emerald-950/80 dark:text-emerald-400"
                                  : "bg-rose-100 text-rose-800 dark:bg-rose-950/80 dark:text-rose-400"
                              }`}
                            >
                              {r.ready ? "Ready" : "Not Ready"}
                            </span>
                          </td>
                          <td className="px-5 py-3">
                            <span className="rounded border border-emerald-700/50 bg-emerald-900/60 px-2 py-0.5 text-[10px] font-bold text-emerald-300">
                              Responsive
                            </span>
                          </td>
                        </>
                      )}

                      {/* CREATED */}
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
          </div>
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
                  {view === "claims"
                    ? selectedResource.claimRef?.kind || selectedResource.kind
                    : selectedResource.kind}
                </span>
                <span className="font-mono text-xs text-zinc-400">
                  {selectedResource.claimRef?.apiVersion || "example.org/v1alpha1"}
                </span>
                <span className="rounded bg-zinc-100 px-2 py-0.5 text-[10px] font-semibold text-zinc-600 dark:bg-zinc-800 dark:text-zinc-300">
                  {view === "claims"
                    ? "Claim"
                    : view === "managed"
                    ? "Managed Resource"
                    : "Composite"}
                </span>
              </div>
              <div className="text-xs text-zinc-500">
                {view === "claims"
                  ? `${selectedResource.claimRef?.namespace || "default"} / ${selectedResource.claimRef?.name || selectedResource.name}`
                  : selectedResource.name}
              </div>
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
                    view={view}
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
  return (
    <div className="space-y-4">
      <div className="text-xs text-zinc-500">
        Manifesto bruto sincronizado do cluster Kubernetes:
      </div>
      <YamlViewer yaml={q.data?.yaml || ""} />
    </div>
  );
}

function StatusTabContent({ resource: r }: { resource: TrackedResource }) {
  const conditions = (r as unknown as { conditions?: { type: string; status: string; reason?: string; message?: string; lastTransitionTime?: string }[] }).conditions || [
    {
      type: "Responsive",
      status: "True",
      reason: "WatchCircuitClosed",
      message: "Resource informer active and responsive in cluster control plane.",
    },
    {
      type: "Synced",
      status: r.synced ? "True" : "False",
      reason: r.synced ? "ReconcileSuccess" : "ReconcileError",
      message: r.synced ? "Successfully reconciled." : "Waiting for reconciliation.",
    },
    {
      type: "Ready",
      status: r.ready ? "True" : "False",
      reason: r.ready ? "Available" : "CreatingOrPending",
      message: r.ready ? "Resource is ready and healthy." : "Resource is not ready yet.",
    },
  ];

  return (
    <div className="space-y-3">
      {conditions.map((c) => {
        const isTrue = c.status === "True";
        return (
          <div
            key={c.type}
            className="flex items-start justify-between rounded-lg border border-zinc-200 bg-zinc-50/50 p-3.5 text-xs dark:border-zinc-800 dark:bg-zinc-900/40"
          >
            <div>
              <div className="flex items-center gap-2">
                <span className="font-semibold text-zinc-900 dark:text-zinc-100">{c.type}</span>
                <span
                  className={`rounded px-1.5 py-0.5 text-[10px] font-bold ${
                    isTrue
                      ? "bg-emerald-100 text-emerald-800 dark:bg-emerald-950/80 dark:text-emerald-400"
                      : "bg-rose-100 text-rose-800 dark:bg-rose-950/80 dark:text-rose-400"
                  }`}
                >
                  {c.status}
                </span>
                {c.reason && (
                  <span className="font-mono text-[11px] text-zinc-400">({c.reason})</span>
                )}
              </div>
              {c.message && (
                <p className="mt-1 text-zinc-600 dark:text-zinc-300">{c.message}</p>
              )}
            </div>
            {c.lastTransitionTime && (
              <span className="text-[10px] text-zinc-400">
                {new Date(c.lastTransitionTime).toLocaleTimeString()}
              </span>
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
  view,
  onOpenLifecycle,
}: {
  resource: TrackedResource;
  view: "claims" | "composites" | "managed";
  onOpenLifecycle?: () => void;
}) {
  return (
    <div className="grid gap-6 md:grid-cols-2">
      <div className="rounded-xl border border-zinc-200 bg-zinc-50/50 p-4 dark:border-zinc-800 dark:bg-zinc-900/30">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-zinc-500">Metadados</h3>
        <dl className="mt-3 grid grid-cols-[8rem_1fr] gap-y-2 text-xs">
          <dt className="text-zinc-400">Owner / Squad</dt>
          <dd className="font-medium text-zinc-900 dark:text-zinc-100">{r.owner || "—"}</dd>

          <dt className="text-zinc-400">Environment</dt>
          <dd>{r.environment || "—"}</dd>

          <dt className="text-zinc-400">Provider</dt>
          <dd className="capitalize">{r.provider || "—"}</dd>

          {view === "claims" && (
            <>
              <dt className="text-zinc-400">Claim Namespace</dt>
              <dd className="font-mono text-[11px]">{r.claimRef?.namespace || "default"}</dd>
              <dt className="text-zinc-400">Claim Name</dt>
              <dd className="font-mono text-[11px]">{r.claimRef?.name || r.name}</dd>
              <dt className="text-zinc-400">Bound Composite</dt>
              <dd className="font-mono text-[11px] text-cyan-500">{r.name}</dd>
            </>
          )}

          {view === "managed" && (
            <>
              <dt className="text-zinc-400">Parent Composite</dt>
              <dd className="font-mono text-[11px]">{r.parentResource || r.claimRef?.name || r.name}</dd>
              <dt className="text-zinc-400">Deletion Policy</dt>
              <dd className="font-mono text-[11px]">{r.deletionPolicy || "Delete"}</dd>
            </>
          )}

          {view === "composites" && (
            <>
              <dt className="text-zinc-400">Claim Ref</dt>
              <dd className="font-mono text-[11px]">{r.claimRef?.namespace}/{r.claimRef?.name}</dd>
              <dt className="text-zinc-400">Composition</dt>
              <dd className="font-mono text-[11px]">{r.composition?.name || "—"}</dd>
            </>
          )}
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
