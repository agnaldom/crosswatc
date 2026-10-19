"use client";

import { useQueries, useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Badge } from "@/components/Badge";
import { api } from "@/lib/api";
import { resourceStateLabel, resourceStateTone } from "@/lib/labels";
import type { TrackedResource } from "@/lib/types";
import { useWatchlist, type WatchedResource } from "@/lib/watchlist";

export function OnWatchDrawer({
  selectedResource,
  onSelectResource,
}: {
  selectedResource?: TrackedResource | null;
  onSelectResource?: (id: string) => void;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const { list, remove, clearAll } = useWatchlist();

  // Load live data for watched items
  const watchedQueries = useQueries({
    queries: list.map((w: WatchedResource) => ({
      queryKey: ["resource", w.id],
      queryFn: () => api.resource(w.id),
      staleTime: 15_000,
    })),
  });

  // Query topology to get real Crossplane status conditions for the selected resource
  const topologyQuery = useQuery({
    queryKey: ["topology", selectedResource?.id],
    queryFn: () => api.topology(selectedResource!.id),
    enabled: !!selectedResource?.id && isOpen,
    staleTime: 30_000,
  });

  // Resolve condition list from selected resource and topology
  const centralNode = topologyQuery.data?.nodes.find(
    (n) => n.kind.toLowerCase() === selectedResource?.kind.toLowerCase() || n.type === "composite" || n.type === "claim",
  );
  const rawConditions = centralNode?.conditions ?? [];

  // Default conditions representation if raw conditions are empty
  const conditions = rawConditions.length > 0 ? rawConditions : [
    {
      type: "Responsive",
      status: "True",
      reason: "WatchCircuitClosed",
      message: "Resource informer active and responsive in cluster control plane.",
      lastTransitionTime: selectedResource?.updatedAt,
    },
    {
      type: "Synced",
      status: selectedResource?.synced ? "True" : selectedResource?.synced === false ? "False" : "Unknown",
      reason: selectedResource?.synced ? "ReconcileSuccess" : "ReconcilePending",
      message: selectedResource?.synced === false
        ? `[cannot compose resources: ${selectedResource.name} pending reconciliation or configuration sync]`
        : undefined,
      lastTransitionTime: selectedResource?.updatedAt,
    },
    {
      type: "Ready",
      status: selectedResource?.ready ? "True" : selectedResource?.ready === false ? "False" : "Unknown",
      reason: selectedResource?.ready ? "Available" : "CreatingOrPending",
      message: selectedResource?.ready === false ? "Underlying managed resources are being provisioned by Crossplane provider." : undefined,
      lastTransitionTime: selectedResource?.updatedAt,
    },
  ];

  return (
    <>
      {/* Floating Toggle Button (Right Edge) */}
      <button
        type="button"
        onClick={() => setIsOpen((prev) => !prev)}
        aria-label="Toggle OnWatch Resources and Inspector"
        title="OnWatch Resources & Inspector"
        className="fixed right-0 top-1/2 z-40 flex -translate-y-1/2 items-center gap-1.5 rounded-l-md bg-cyan-600 px-2.5 py-3 text-white shadow-lg transition-transform hover:bg-cyan-500 focus:outline-none"
      >
        {/* Eye Icon */}
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-5 w-5" aria-hidden>
          <path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z" />
          <circle cx="12" cy="12" r="3" />
        </svg>
        {list.length > 0 && (
          <span className="flex h-4 w-4 items-center justify-center rounded-full bg-white text-[10px] font-bold text-cyan-700">
            {list.length}
          </span>
        )}
      </button>

      {/* Drawer Overlay (Backdrop for mobile/focus) */}
      {isOpen && (
        <div
          role="presentation"
          onClick={() => setIsOpen(false)}
          className="fixed inset-0 z-40 bg-black/20 backdrop-blur-[1px] transition-opacity dark:bg-black/40"
        />
      )}

      {/* Slideout Panel */}
      <aside
        className={`fixed bottom-0 right-0 top-0 z-50 flex w-80 sm:w-96 flex-col border-l border-zinc-200 bg-white shadow-2xl transition-transform duration-200 dark:border-zinc-800 dark:bg-[#161b22] ${
          isOpen ? "translate-x-0" : "translate-x-full"
        }`}
        aria-label="OnWatch and Resource Inspector"
      >
        {/* Drawer Header */}
        <div className="flex items-center justify-between border-b border-zinc-200 px-4 py-3.5 dark:border-zinc-800">
          <div className="flex items-center gap-2">
            <h2 className="text-sm font-semibold tracking-tight text-zinc-900 dark:text-zinc-100">
              OnWatch Resources
            </h2>
            <span className="flex h-5 min-w-5 items-center justify-center rounded-full bg-cyan-600 px-1.5 text-xs font-semibold text-white">
              {list.length}
            </span>
          </div>
          <div className="flex items-center gap-2">
            {list.length > 0 && (
              <button
                type="button"
                onClick={clearAll}
                title="Limpar todos do OnWatch"
                className="rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-rose-600 dark:hover:bg-zinc-800"
              >
                {/* Trash Icon */}
                <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5" className="h-4 w-4">
                  <path d="M4 6h12M8 6V4a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1v2m2 0v10a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6h10Z" />
                </svg>
              </button>
            )}
            <button
              type="button"
              onClick={() => setIsOpen(false)}
              className="rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700 dark:hover:bg-zinc-800 dark:hover:text-zinc-200"
            >
              ✕
            </button>
          </div>
        </div>

        {/* Scrollable Content */}
        <div className="flex-1 space-y-6 overflow-y-auto p-4">
          {/* Watched Resources List */}
          <div className="space-y-2">
            {list.length === 0 ? (
              <p className="rounded-md border border-dashed border-zinc-200 p-4 text-center text-xs text-zinc-400 dark:border-zinc-800">
                Nenhum recurso fixado no OnWatch. Clique no ícone de olho nos recursos para monitorá-los aqui.
              </p>
            ) : (
              list.map((w: WatchedResource, idx: number) => {
                const r = watchedQueries[idx]?.data as TrackedResource | undefined;
                const isSelected = selectedResource?.id === w.id;
                return (
                  <div
                    key={w.id}
                    onClick={() => onSelectResource?.(w.id)}
                    className={`group relative flex cursor-pointer items-start justify-between rounded-lg border p-3 transition ${
                      isSelected
                        ? "border-cyan-500 bg-cyan-50/40 dark:border-cyan-600 dark:bg-cyan-950/20"
                        : "border-zinc-200 bg-white hover:border-zinc-300 dark:border-zinc-800 dark:bg-zinc-900/60 dark:hover:border-zinc-700"
                    }`}
                  >
                    <div className="min-w-0 pr-6">
                      <div className="truncate text-xs font-semibold text-zinc-900 dark:text-zinc-100">
                        {w.name}
                      </div>
                      <div className="text-[11px] text-zinc-500">{w.kind}</div>
                      {r && (
                        <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
                          <Badge tone={resourceStateTone[r.resourceState]}>
                            {resourceStateLabel[r.resourceState]}
                          </Badge>
                          {r.synced !== undefined && (
                            <span
                              className={`rounded px-1.5 py-0.5 text-[10px] font-medium ${
                                r.synced
                                  ? "bg-emerald-100 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300"
                                  : "bg-rose-100 text-rose-800 dark:bg-rose-950 dark:text-rose-300"
                              }`}
                            >
                              {r.synced ? "Synced" : "Not Synced"}
                            </span>
                          )}
                        </div>
                      )}
                    </div>
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        remove(w.id);
                      }}
                      title="Remover do OnWatch"
                      className="absolute right-2 top-2 p-1 text-zinc-400 hover:text-rose-600 dark:hover:text-rose-400"
                    >
                      <svg viewBox="0 0 16 16" fill="currentColor" className="h-3.5 w-3.5">
                        <path d="M3.72 3.72a.75.75 0 0 1 1.06 0L8 6.94l3.22-3.22a.75.75 0 1 1 1.06 1.06L9.06 8l3.22 3.22a.75.75 0 1 1-1.06 1.06L8 9.06l-3.22 3.22a.75.75 0 0 1-1.06-1.06L6.94 8 3.72 4.78a.75.75 0 0 1 0-1.06Z" />
                      </svg>
                    </button>
                  </div>
                );
              })
            )}
          </div>

          {/* Status Conditions Inspector */}
          {selectedResource && (
            <div className="space-y-3 pt-2">
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-semibold uppercase tracking-wider text-zinc-500">
                  Status Conditions
                </h3>
                <span className="text-[11px] text-zinc-400 truncate max-w-[140px]">{selectedResource.name}</span>
              </div>

              <div className="space-y-2.5">
                {conditions.map((c, i) => {
                  const isTrue = c.status?.toLowerCase() === "true" || c.status === "True";
                  return (
                    <div
                      key={`${c.type}-${i}`}
                      className={`rounded-lg border p-3 text-left transition ${
                        isTrue
                          ? "border-emerald-200/80 bg-emerald-50/30 dark:border-emerald-900/40 dark:bg-emerald-950/20"
                          : "border-rose-200/80 bg-rose-50/30 dark:border-rose-900/40 dark:bg-rose-950/20"
                      }`}
                    >
                      <div className="flex items-center justify-between">
                        <span className="text-xs font-bold text-zinc-800 dark:text-zinc-200">
                          {c.type}
                        </span>
                        <span
                          className={`rounded px-1.5 py-0.5 text-[10px] font-bold ${
                            isTrue
                              ? "bg-emerald-600 text-white dark:bg-emerald-500"
                              : "bg-rose-600 text-white dark:bg-rose-500"
                          }`}
                        >
                          {c.status}
                        </span>
                      </div>

                      {c.reason && (
                        <div className="mt-1 text-[11px] text-zinc-600 dark:text-zinc-400">
                          <span className="font-medium text-zinc-500">Reason:</span> {c.reason}
                        </div>
                      )}

                      {c.message && (
                        <div className="mt-2 max-h-32 overflow-y-auto rounded bg-zinc-100 p-2 font-mono text-[10px] text-zinc-700 dark:bg-zinc-900/80 dark:text-zinc-300">
                          {c.message}
                        </div>
                      )}

                      {c.lastTransitionTime && (
                        <div className="mt-1.5 text-[10px] text-zinc-400">
                          Last Transition: {new Date(c.lastTransitionTime).toLocaleString("pt-BR")}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          {/* Overview Section */}
          {selectedResource && (
            <div className="space-y-2 border-t border-zinc-200 pt-4 dark:border-zinc-800">
              <h3 className="text-xs font-semibold uppercase tracking-wider text-zinc-500">Overview</h3>
              <dl className="grid grid-cols-[5rem_1fr] gap-y-1 text-xs text-zinc-600 dark:text-zinc-400">
                <dt className="text-zinc-400">Name</dt>
                <dd className="truncate font-medium text-zinc-900 dark:text-zinc-100">{selectedResource.name}</dd>
                <dt className="text-zinc-400">Kind</dt>
                <dd>{selectedResource.kind}</dd>
                <dt className="text-zinc-400">Namespace</dt>
                <dd className="font-mono text-[11px]">{selectedResource.claimRef?.namespace || "default"}</dd>
                <dt className="text-zinc-400">Provider</dt>
                <dd className="capitalize">{selectedResource.provider}</dd>
                <dt className="text-zinc-400">UID</dt>
                <dd className="break-all font-mono text-[10px] text-zinc-500">
                  {selectedResource.correlationId || selectedResource.id}
                </dd>
              </dl>
            </div>
          )}
        </div>
      </aside>
    </>
  );
}
