"use client";

import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { Badge } from "@/components/Badge";
import { ErrorState, Loading } from "@/components/States";
import { api } from "@/lib/api";

const key = (o: { kind: string; name: string }) => `${o.kind}/${o.name}`;

export function KubeEvents({
  id,
  filter,
  onFilter,
}: {
  id: string;
  filter: string;
  onFilter: (f: string) => void;
}) {
  const [search, setSearch] = useState("");
  const [typeFilter, setTypeFilter] = useState<"all" | "Warning" | "Normal">("all");
  const [copiedIdx, setCopiedIdx] = useState<number | null>(null);

  const q = useQuery({ queryKey: ["kube-events", id], queryFn: () => api.kubeEvents(id) });

  const rawItems = useMemo(() => q.data?.items ?? [], [q.data?.items]);

  const objects = useMemo(() => {
    return [...new Set(rawItems.map((e) => key(e.object)))].sort();
  }, [rawItems]);

  const stats = useMemo(() => {
    const total = rawItems.length;
    const warnings = rawItems.filter((e) => e.type === "Warning").length;
    const normal = rawItems.filter((e) => e.type === "Normal").length;
    return { total, warnings, normal };
  }, [rawItems]);

  const filteredItems = useMemo(() => {
    return rawItems.filter((e) => {
      // Object filter
      if (filter && key(e.object) !== filter) {
        return false;
      }

      // Severity type filter
      if (typeFilter !== "all" && e.type !== typeFilter) {
        return false;
      }

      // Search text filter
      if (search) {
        const query = search.toLowerCase();
        const matchesMsg = e.message.toLowerCase().includes(query);
        const matchesReason = e.reason.toLowerCase().includes(query);
        const matchesObj = key(e.object).toLowerCase().includes(query);
        const matchesSrc = e.source?.toLowerCase().includes(query) ?? false;
        if (!matchesMsg && !matchesReason && !matchesObj && !matchesSrc) {
          return false;
        }
      }

      return true;
    });
  }, [rawItems, filter, typeFilter, search]);

  const handleCopyMessage = (msg: string, idx: number) => {
    navigator.clipboard.writeText(msg);
    setCopiedIdx(idx);
    setTimeout(() => setCopiedIdx(null), 2000);
  };

  if (q.isPending) return <Loading />;
  if (q.isError) return <ErrorState error={q.error} />;

  return (
    <div className="space-y-4">
      {/* Controls Bar */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2.5">
          {/* Object Selector */}
          <div className="flex items-center gap-1.5 text-xs text-zinc-600 dark:text-zinc-400">
            <label htmlFor="obj" className="font-medium">
              Object
            </label>
            <select
              id="obj"
              aria-label="Object"
              value={filter}
              onChange={(e) => onFilter(e.target.value)}
              className="h-8 rounded-md border border-zinc-200 bg-white px-2.5 text-xs font-medium text-zinc-700 shadow-sm transition hover:bg-zinc-50 focus:border-zinc-400 focus:outline-none dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:bg-zinc-800/60"
            >
              <option value="">All (Claim, XR, and MRs)</option>
              {objects.map((o) => (
                <option key={o} value={o}>
                  {o}
                </option>
              ))}
              {filter && !objects.includes(filter) && <option value={filter}>{filter}</option>}
            </select>
          </div>

          {/* Severity Filter */}
          <select
            aria-label="Filter by Type"
            value={typeFilter}
            onChange={(e) => setTypeFilter(e.target.value as "all" | "Warning" | "Normal")}
            className="h-8 rounded-md border border-zinc-200 bg-white px-2.5 text-xs font-medium text-zinc-700 shadow-sm transition hover:bg-zinc-50 focus:border-zinc-400 focus:outline-none dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:bg-zinc-800/60"
          >
            <option value="all">All Types</option>
            <option value="Warning">Warning Only</option>
            <option value="Normal">Normal Only</option>
          </select>

          {/* Search Box */}
          <div className="relative">
            <svg
              viewBox="0 0 20 20"
              fill="currentColor"
              className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-zinc-400"
            >
              <path
                fillRule="evenodd"
                d="M9 3.5a5.5 5.5 0 100 11 5.5 5.5 0 000-11zM2 9a7 7 0 1112.452 4.391l3.328 3.329a.75.75 0 11-1.06 1.06l-3.329-3.328A7 7 0 012 9z"
                clipRule="evenodd"
              />
            </svg>
            <input
              type="text"
              aria-label="Search events"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search events..."
              className="h-8 w-48 rounded-md border border-zinc-200 bg-white pl-8 pr-3 text-xs text-zinc-900 shadow-sm transition placeholder:text-zinc-400 focus:border-zinc-400 focus:outline-none focus:ring-1 focus:ring-zinc-400 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100"
            />
          </div>
        </div>

        {/* Counters & Stats Badges */}
        <div className="flex items-center gap-1.5 text-xs">
          <span className="inline-flex items-center rounded-md bg-zinc-100 px-2 py-0.5 text-[11px] font-medium text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400">
            Total: {stats.total}
          </span>
          {stats.warnings > 0 && (
            <span className="inline-flex items-center rounded-md bg-red-50 px-2 py-0.5 text-[11px] font-semibold text-red-700 dark:bg-red-950/60 dark:text-red-400">
              {stats.warnings} Warnings
            </span>
          )}
          <span className="inline-flex items-center rounded-md bg-emerald-50 px-2 py-0.5 text-[11px] font-semibold text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400">
            {stats.normal} Normal
          </span>
        </div>
      </div>

      {/* Cluster warning notice if any error reading from objects */}
      {q.data.errors && q.data.errors.length > 0 && (
        <div className="rounded-lg border border-amber-200 bg-amber-50/80 p-2.5 text-xs text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-200">
          Could not read events from:{" "}
          {q.data.errors.map((e) => `${key(e.object)} (${e.error})`).join("; ")}
        </div>
      )}

      {/* Events List */}
      {filteredItems.length === 0 ? (
        <div className="flex h-36 flex-col items-center justify-center rounded-xl border border-dashed border-zinc-200 bg-zinc-50/50 p-6 text-center dark:border-zinc-800 dark:bg-zinc-900/30">
          <p className="text-sm font-semibold text-zinc-700 dark:text-zinc-300">
            {search || typeFilter !== "all"
              ? "No events matched your search criteria."
              : `No Kubernetes events${filter ? ` for ${filter}` : ""}. Events expire in the cluster (default: 1 h).`}
          </p>
        </div>
      ) : (
        <ol className="divide-y divide-zinc-100 overflow-hidden rounded-xl border border-zinc-200 bg-white shadow-xs dark:divide-zinc-800/60 dark:border-zinc-800 dark:bg-zinc-900">
          {filteredItems.map((e, i) => (
            <li
              key={i}
              className="flex flex-col gap-2 p-3.5 transition hover:bg-zinc-50/60 sm:flex-row sm:items-start sm:gap-4 dark:hover:bg-zinc-800/30"
            >
              {/* Timestamp */}
              <time
                className="w-36 shrink-0 font-mono text-[11px] text-zinc-500 dark:text-zinc-400"
                dateTime={e.lastSeen}
                title={e.lastSeen}
              >
                {e.lastSeen ? new Date(e.lastSeen).toLocaleString("en-US") : "—"}
              </time>

              {/* Event Body */}
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <Badge tone={e.type === "Warning" ? "error" : "ok"}>{e.type}</Badge>
                  <span className="font-semibold text-zinc-900 dark:text-zinc-100">{e.reason}</span>
                  {e.count > 1 && (
                    <span className="rounded bg-zinc-100 px-1.5 py-0.2 font-mono text-[10px] text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400">
                      ×{e.count}
                    </span>
                  )}
                  <span className="rounded bg-zinc-100 px-1.5 py-0.5 font-mono text-[11px] text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300">
                    {key(e.object)}
                  </span>
                  {e.object.namespace && (
                    <span className="text-[10px] text-zinc-400 dark:text-zinc-500">
                      ns: {e.object.namespace}
                    </span>
                  )}
                </div>

                <div className="mt-1 flex items-start justify-between gap-2">
                  <p className="text-xs text-zinc-700 break-words dark:text-zinc-300">{e.message}</p>
                  <button
                    type="button"
                    onClick={() => handleCopyMessage(e.message, i)}
                    className="shrink-0 text-[10px] text-zinc-400 transition hover:text-zinc-700 dark:hover:text-zinc-200"
                    title="Copy message"
                  >
                    {copiedIdx === i ? "Copied!" : "Copy"}
                  </button>
                </div>

                {e.source && (
                  <p className="mt-1 text-[11px] text-zinc-400 dark:text-zinc-500">
                    Source: <span className="font-mono">{e.source}</span>
                  </p>
                )}
              </div>
            </li>
          ))}
        </ol>
      )}
    </div>
  );
}
