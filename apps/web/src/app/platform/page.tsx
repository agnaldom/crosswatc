"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useMemo, useState } from "react";
import { Badge } from "@/components/Badge";
import { ErrorState, Loading } from "@/components/States";
import { api } from "@/lib/api";
import { normalizeConditions } from "@/lib/conditions";
import type { CrossplanePackage } from "@/lib/types";

const tabs = ["Providers", "Functions", "Configurations", "XRDs", "Compositions", "MRDs", "Activation Policies"] as const;
type Tab = (typeof tabs)[number];

export default function PlatformPage() {
  const [tab, setTab] = useState<Tab>("Providers");
  const me = useQuery({ queryKey: ["me"], queryFn: api.me, refetchInterval: false });

  if (me.isPending) return <Loading />;
  if (me.isError) return <ErrorState error={me.error} />;
  if (!me.data.sre) {
    return (
      <div className="rounded-xl border border-amber-200 bg-amber-50/80 p-4 text-sm text-amber-900 shadow-sm dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-200">
        O catálogo de plataforma é restrito ao time de SRE/Plataforma.
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-7xl space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-100">
          Platform Catalog
        </h1>
        <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
          Crossplane control plane installation health, packages, XRDs, and compositions.
        </p>
      </div>

      {/* Tabs navigation */}
      <nav
        className="flex flex-wrap items-center gap-1 rounded-xl border border-zinc-200 bg-zinc-100/80 p-1 dark:border-zinc-800 dark:bg-zinc-900/80"
        role="tablist"
      >
        {tabs.map((t) => (
          <button
            key={t}
            role="tab"
            aria-selected={tab === t}
            type="button"
            onClick={() => setTab(t)}
            className={`flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-semibold transition ${
              tab === t
                ? "bg-white text-zinc-900 shadow-sm dark:bg-zinc-800 dark:text-white"
                : "text-zinc-500 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white"
            }`}
          >
            <span>{t}</span>
          </button>
        ))}
      </nav>

      {/* Tab Panels */}
      <section className="mt-4">
        {tab === "Providers" && <PackagesSection kind="providers" title="Providers" />}
        {tab === "Functions" && <PackagesSection kind="functions" title="Functions" />}
        {tab === "Configurations" && <PackagesSection kind="configurations" title="Configurations" />}
        {tab === "XRDs" && <XRDsSection />}
        {tab === "Compositions" && <CompositionsSection />}
        {tab === "MRDs" && <MRDsSection />}
        {tab === "Activation Policies" && <MRAPsSection />}
      </section>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Packages (Providers / Functions / Configurations)
// ---------------------------------------------------------------------------

function PackagesSection({
  kind,
  title,
}: {
  kind: "providers" | "functions" | "configurations";
  title: string;
}) {
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [viewMode, setViewMode] = useState<"cards" | "table">("cards");

  const q = useQuery({
    queryKey: ["platform", kind],
    queryFn: () => api.platformPackages(kind),
  });

  const allItems = useMemo(() => q.data?.items ?? [], [q.data?.items]);

  const filteredItems = useMemo(() => {
    return allItems.filter((p) => {
      // Search matching name or package
      if (search) {
        const query = search.toLowerCase();
        const matchesName = p.name.toLowerCase().includes(query);
        const matchesPkg = p.package.toLowerCase().includes(query);
        if (!matchesName && !matchesPkg) return false;
      }

      // Status filtering
      if (statusFilter !== "all") {
        const healthyCond = p.conditions.find((c) => c.type === "Healthy");
        const installedCond = p.conditions.find((c) => c.type === "Installed");

        const isHealthy = p.healthy ?? (healthyCond?.status === "True");
        const isInstalled = p.installed ?? (installedCond?.status === "True");

        if (statusFilter === "healthy" && !isHealthy) return false;
        if (statusFilter === "unhealthy" && (isHealthy || !isInstalled)) return false;
        if (statusFilter === "installed" && !isInstalled) return false;
        if (statusFilter === "not-installed" && isInstalled) return false;
      }

      return true;
    });
  }, [allItems, search, statusFilter]);

  if (q.isPending) return <Loading />;
  if (q.isError) return <ErrorState error={q.error} />;

  return (
    <div className="space-y-4">
      {/* Controls Bar */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2.5">
          {/* Search box */}
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
              aria-label={`Search ${title}`}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search..."
              className="h-8 w-56 rounded-md border border-zinc-200 bg-white pl-9 pr-3 text-xs text-zinc-900 shadow-sm transition placeholder:text-zinc-400 focus:border-zinc-400 focus:outline-none focus:ring-1 focus:ring-zinc-400 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100"
            />
          </div>

          {/* Status Dropdown */}
          <select
            aria-label="Filter by Status"
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="h-8 rounded-md border border-zinc-200 bg-white px-2.5 text-xs font-medium text-zinc-700 shadow-sm transition hover:bg-zinc-50 focus:border-zinc-400 focus:outline-none dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:bg-zinc-800/60"
          >
            <option value="all">All Status</option>
            <option value="healthy">Healthy</option>
            <option value="unhealthy">Unhealthy</option>
            <option value="installed">Installed</option>
            <option value="not-installed">Not Installed</option>
          </select>

          {/* View mode toggle */}
          <div className="flex items-center rounded-md border border-zinc-200 bg-zinc-100 p-0.5 dark:border-zinc-800 dark:bg-zinc-900">
            <button
              type="button"
              aria-label="Cards view"
              onClick={() => setViewMode("cards")}
              className={`rounded p-1 transition ${
                viewMode === "cards"
                  ? "bg-white text-zinc-900 shadow-xs dark:bg-zinc-800 dark:text-white"
                  : "text-zinc-500 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white"
              }`}
            >
              <svg viewBox="0 0 20 20" fill="currentColor" className="h-4 w-4">
                <path d="M5 3a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2H5zM5 11a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2v-2a2 2 0 00-2-2H5zM11 5a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V5zM11 13a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
              </svg>
            </button>
            <button
              type="button"
              aria-label="Table view"
              onClick={() => setViewMode("table")}
              className={`rounded p-1 transition ${
                viewMode === "table"
                  ? "bg-white text-zinc-900 shadow-xs dark:bg-zinc-800 dark:text-white"
                  : "text-zinc-500 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white"
              }`}
            >
              <svg viewBox="0 0 20 20" fill="currentColor" className="h-4 w-4">
                <path
                  fillRule="evenodd"
                  d="M2 4.75A.75.75 0 012.75 4h14.5a.75.75 0 010 1.5H2.75A.75.75 0 012 4.75zm0 5A.75.75 0 012.75 9h14.5a.75.75 0 010 1.5H2.75A.75.75 0 012 9.75zm0 5a.75.75 0 012.75-.75h14.5a.75.75 0 010 1.5H2.75a.75.75 0 01-.75-.75z"
                  clipRule="evenodd"
                />
              </svg>
            </button>
          </div>
        </div>

        {/* Counter */}
        <span className="text-xs font-medium text-zinc-500 dark:text-zinc-400">
          {filteredItems.length} results
        </span>
      </div>

      {/* Main Content */}
      {filteredItems.length === 0 ? (
        <div className="flex h-48 flex-col items-center justify-center rounded-xl border border-dashed border-zinc-200 bg-zinc-50/50 p-8 text-center dark:border-zinc-800 dark:bg-zinc-900/30">
          <p className="text-sm font-semibold text-zinc-700 dark:text-zinc-300">No data found</p>
          <p className="mt-1 text-xs text-zinc-500 dark:text-zinc-400">
            No packages matched the search or filter criteria.
          </p>
        </div>
      ) : viewMode === "cards" ? (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {filteredItems.map((p) => (
            <PackageCard key={p.name} p={p} />
          ))}
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white shadow-xs dark:border-zinc-800 dark:bg-zinc-900">
          <table className="w-full text-left text-xs">
            <thead className="border-b border-zinc-200 bg-zinc-50/80 text-[11px] font-semibold uppercase tracking-wider text-zinc-500 dark:border-zinc-800 dark:bg-zinc-800/40 dark:text-zinc-400">
              <tr>
                <th className="px-4 py-3">NAME</th>
                <th className="px-4 py-3">HEALTH</th>
                <th className="px-4 py-3">INSTALLED</th>
                <th className="px-4 py-3">PACKAGE</th>
                <th className="px-4 py-3">REVISION</th>
                <th className="px-4 py-3">ACTIVATION</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-100 dark:divide-zinc-800/60">
              {filteredItems.map((p) => (
                <PackageTableRow key={p.name} p={p} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function PackageCard({ p }: { p: CrossplanePackage }) {
  const healthyCond = p.conditions.find((c) => c.type === "Healthy");
  const installedCond = p.conditions.find((c) => c.type === "Installed");
  const isHealthy = p.healthy ?? (healthyCond?.status === "True");
  const isInstalled = p.installed ?? (installedCond?.status === "True");

  const errorReason = healthyCond?.status === "False" ? healthyCond.message || healthyCond.reason : null;

  return (
    <div className="flex flex-col justify-between rounded-xl border border-zinc-200 bg-white p-4 shadow-xs transition hover:border-zinc-300 dark:border-zinc-800 dark:bg-zinc-900 dark:hover:border-zinc-700">
      <div className="space-y-3">
        {/* Title and Badges */}
        <div className="flex items-start justify-between gap-2">
          <div>
            <h3 className="font-semibold text-zinc-900 dark:text-zinc-100">{p.name}</h3>
            <span className="text-[11px] font-medium text-zinc-500 dark:text-zinc-400">{p.kind}</span>
          </div>

          <div className="flex flex-wrap items-center gap-1.5">
            <span
              className={`inline-flex items-center rounded-md px-2 py-0.5 text-[11px] font-semibold ${
                isInstalled
                  ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400"
                  : "bg-red-50 text-red-700 dark:bg-red-950/60 dark:text-red-400"
              }`}
            >
              {isInstalled ? "Installed" : "Not Installed"}
            </span>

            <span
              className={`inline-flex items-center rounded-md px-2 py-0.5 text-[11px] font-semibold ${
                isHealthy
                  ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400"
                  : "bg-red-50 text-red-700 dark:bg-red-950/60 dark:text-red-400"
              }`}
            >
              {isHealthy ? "Healthy" : "Unhealthy"}
            </span>
          </div>
        </div>

        {/* Package Reference */}
        <div>
          <label className="text-[10px] font-semibold uppercase tracking-wider text-zinc-400">OCI Package</label>
          <div className="mt-0.5 rounded-md bg-zinc-50 p-2 font-mono text-[11px] text-zinc-700 break-all dark:bg-zinc-800/60 dark:text-zinc-300">
            {p.package}
          </div>
        </div>

        {/* Error message detail if unhealthy */}
        {errorReason && (
          <div className="rounded-md border border-red-200 bg-red-50/70 p-2 text-xs text-red-800 dark:border-red-900/50 dark:bg-red-950/40 dark:text-red-300">
            {errorReason}
          </div>
        )}
      </div>

      {/* Footer Info */}
      <div className="mt-4 flex items-center justify-between border-t border-zinc-100 pt-3 text-[11px] text-zinc-500 dark:border-zinc-800 dark:text-zinc-400">
        <span className="font-mono">Rev: {p.currentRevision ?? "—"}</span>
        <span>{p.revisionActivationPolicy ?? "Automatic"}</span>
      </div>
    </div>
  );
}

function PackageTableRow({ p }: { p: CrossplanePackage }) {
  const healthyCond = p.conditions.find((c) => c.type === "Healthy");
  const installedCond = p.conditions.find((c) => c.type === "Installed");
  const isHealthy = p.healthy ?? (healthyCond?.status === "True");
  const isInstalled = p.installed ?? (installedCond?.status === "True");

  return (
    <tr className="transition hover:bg-zinc-50/60 dark:hover:bg-zinc-800/30">
      <td className="px-4 py-3 font-medium text-zinc-900 dark:text-zinc-100">{p.name}</td>
      <td className="px-4 py-3">
        <span
          className={`inline-flex items-center rounded-md px-2 py-0.5 text-[11px] font-semibold ${
            isHealthy
              ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400"
              : "bg-red-50 text-red-700 dark:bg-red-950/60 dark:text-red-400"
          }`}
        >
          {isHealthy ? "Healthy" : "Unhealthy"}
        </span>
      </td>
      <td className="px-4 py-3">
        <span
          className={`inline-flex items-center rounded-md px-2 py-0.5 text-[11px] font-semibold ${
            isInstalled
              ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400"
              : "bg-red-50 text-red-700 dark:bg-red-950/60 dark:text-red-400"
          }`}
        >
          {isInstalled ? "Installed" : "Not Installed"}
        </span>
      </td>
      <td className="px-4 py-3 font-mono text-[11px] text-zinc-600 break-all dark:text-zinc-400">{p.package}</td>
      <td className="px-4 py-3 font-mono text-[11px] text-zinc-600 dark:text-zinc-400">{p.currentRevision ?? "—"}</td>
      <td className="px-4 py-3 text-zinc-600 dark:text-zinc-400">{p.revisionActivationPolicy ?? "Automatic"}</td>
    </tr>
  );
}

// ---------------------------------------------------------------------------
// XRDs (Composite Resource Definitions)
// ---------------------------------------------------------------------------

function XRDsSection() {
  const [search, setSearch] = useState("");
  const [groupFilter, setGroupFilter] = useState("all");

  const q = useQuery({ queryKey: ["platform", "xrds"], queryFn: api.platformXRDs });
  const allItems = useMemo(() => q.data?.items ?? [], [q.data?.items]);

  const groups = useMemo(() => {
    return Array.from(new Set(allItems.map((x) => x.group).filter(Boolean))).sort();
  }, [allItems]);

  const filteredItems = useMemo(() => {
    return allItems.filter((x) => {
      if (search) {
        const query = search.toLowerCase();
        const matchesName = x.name.toLowerCase().includes(query);
        const matchesKind = x.kind.toLowerCase().includes(query);
        const matchesClaim = (x.claimKind || "").toLowerCase().includes(query);
        if (!matchesName && !matchesKind && !matchesClaim) return false;
      }

      if (groupFilter !== "all" && x.group !== groupFilter) {
        return false;
      }

      return true;
    });
  }, [allItems, search, groupFilter]);

  if (q.isPending) return <Loading />;
  if (q.isError) return <ErrorState error={q.error} />;

  return (
    <div className="space-y-4">
      {/* Controls Bar */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2.5">
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
              aria-label="Search XRDs"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search XRDs..."
              className="h-8 w-56 rounded-md border border-zinc-200 bg-white pl-9 pr-3 text-xs text-zinc-900 shadow-sm transition placeholder:text-zinc-400 focus:border-zinc-400 focus:outline-none focus:ring-1 focus:ring-zinc-400 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100"
            />
          </div>

          <select
            aria-label="Filter by Group"
            value={groupFilter}
            onChange={(e) => setGroupFilter(e.target.value)}
            className="h-8 rounded-md border border-zinc-200 bg-white px-2.5 text-xs font-medium text-zinc-700 shadow-sm transition hover:bg-zinc-50 focus:border-zinc-400 focus:outline-none dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:bg-zinc-800/60"
          >
            <option value="all">All Groups</option>
            {groups.map((g) => (
              <option key={g} value={g}>
                {g}
              </option>
            ))}
          </select>
        </div>

        <span className="text-xs font-medium text-zinc-500 dark:text-zinc-400">
          {filteredItems.length} results
        </span>
      </div>

      {filteredItems.length === 0 ? (
        <div className="flex h-48 flex-col items-center justify-center rounded-xl border border-dashed border-zinc-200 bg-zinc-50/50 p-8 text-center dark:border-zinc-800 dark:bg-zinc-900/30">
          <p className="text-sm font-semibold text-zinc-700 dark:text-zinc-300">No data found</p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white shadow-xs dark:border-zinc-800 dark:bg-zinc-900">
          <table className="w-full text-left text-xs">
            <thead className="border-b border-zinc-200 bg-zinc-50/80 text-[11px] font-semibold uppercase tracking-wider text-zinc-500 dark:border-zinc-800 dark:bg-zinc-800/40 dark:text-zinc-400">
              <tr>
                <th className="px-4 py-3">NAME</th>
                <th className="px-4 py-3">GROUP</th>
                <th className="px-4 py-3">RESOURCE KIND</th>
                <th className="px-4 py-3">CLAIM KIND</th>
                <th className="px-4 py-3">STATUS</th>
                <th className="px-4 py-3">SCOPE</th>
                <th className="px-4 py-3">VERSION</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-100 dark:divide-zinc-800/60">
              {filteredItems.map((x) => (
                <tr key={x.name} className="transition hover:bg-zinc-50/60 dark:hover:bg-zinc-800/30">
                  <td className="px-4 py-3 font-mono text-[11px] font-medium text-zinc-900 dark:text-zinc-100">
                    {x.name}
                  </td>
                  <td className="px-4 py-3 font-mono text-[11px] text-zinc-600 dark:text-zinc-400">{x.group}</td>
                  <td className="px-4 py-3 font-medium text-zinc-800 dark:text-zinc-200">
                    <Link
                      href={`/infrastructure?view=composites&kind=${encodeURIComponent(x.kind)}`}
                      className="inline-flex items-center gap-1 font-semibold text-cyan-600 hover:text-cyan-500 hover:underline dark:text-cyan-400"
                      title={`View all ${x.kind} Composite Resources`}
                    >
                      <span>{x.kind}</span>
                      <svg className="h-3 w-3 opacity-60" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                      </svg>
                    </Link>
                  </td>
                  <td className="px-4 py-3 text-zinc-600 dark:text-zinc-400">
                    {x.claimKind ? (
                      <Link
                        href={`/infrastructure?view=claims&kind=${encodeURIComponent(x.claimKind)}`}
                        className="inline-flex items-center gap-1 font-semibold text-cyan-600 hover:text-cyan-500 hover:underline dark:text-cyan-400"
                        title={`View all ${x.claimKind} Claims`}
                      >
                        <span>{x.claimKind}</span>
                        <svg className="h-3 w-3 opacity-60" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                        </svg>
                      </Link>
                    ) : (
                      "—"
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-1.5">
                      <span
                        className={`inline-flex items-center rounded-md px-2 py-0.5 text-[11px] font-semibold ${
                          x.established
                            ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400"
                            : "bg-red-50 text-red-700 dark:bg-red-950/60 dark:text-red-400"
                        }`}
                      >
                        {x.established ? "Established" : "Not Established"}
                      </span>
                      {x.offered !== undefined && (
                        <span
                          className={`inline-flex items-center rounded-md px-2 py-0.5 text-[11px] font-semibold ${
                            x.offered
                              ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400"
                              : "bg-amber-50 text-amber-700 dark:bg-amber-950/60 dark:text-amber-400"
                          }`}
                        >
                          {x.offered ? "Offered" : "Not Offered"}
                        </span>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-zinc-600 dark:text-zinc-400">{x.scope}</td>
                  <td className="px-4 py-3 font-mono text-[11px] text-zinc-600 dark:text-zinc-400">{x.version}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Compositions
// ---------------------------------------------------------------------------

function CompositionsSection() {
  const [search, setSearch] = useState("");

  const q = useQuery({ queryKey: ["platform", "compositions"], queryFn: api.platformCompositions });
  const allItems = useMemo(() => q.data?.items ?? [], [q.data?.items]);

  const filteredItems = useMemo(() => {
    if (!search) return allItems;
    const query = search.toLowerCase();
    return allItems.filter(
      (c) =>
        c.name.toLowerCase().includes(query) ||
        c.compositeType.toLowerCase().includes(query) ||
        (c.functions && c.functions.some((f) => f.toLowerCase().includes(query)))
    );
  }, [allItems, search]);

  if (q.isPending) return <Loading />;
  if (q.isError) return <ErrorState error={q.error} />;

  return (
    <div className="space-y-4">
      {/* Search Bar */}
      <div className="flex flex-wrap items-center justify-between gap-3">
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
            aria-label="Search Compositions"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search compositions..."
            className="h-8 w-56 rounded-md border border-zinc-200 bg-white pl-9 pr-3 text-xs text-zinc-900 shadow-sm transition placeholder:text-zinc-400 focus:border-zinc-400 focus:outline-none focus:ring-1 focus:ring-zinc-400 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100"
          />
        </div>

        <span className="text-xs font-medium text-zinc-500 dark:text-zinc-400">
          {filteredItems.length} results
        </span>
      </div>

      {filteredItems.length === 0 ? (
        <div className="flex h-48 flex-col items-center justify-center rounded-xl border border-dashed border-zinc-200 bg-zinc-50/50 p-8 text-center dark:border-zinc-800 dark:bg-zinc-900/30">
          <p className="text-sm font-semibold text-zinc-700 dark:text-zinc-300">No data found</p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white shadow-xs dark:border-zinc-800 dark:bg-zinc-900">
          <table className="w-full text-left text-xs">
            <thead className="border-b border-zinc-200 bg-zinc-50/80 text-[11px] font-semibold uppercase tracking-wider text-zinc-500 dark:border-zinc-800 dark:bg-zinc-800/40 dark:text-zinc-400">
              <tr>
                <th className="px-4 py-3">NAME</th>
                <th className="px-4 py-3">XR KIND</th>
                <th className="px-4 py-3">MODE</th>
                <th className="px-4 py-3">STEPS</th>
                <th className="px-4 py-3">FUNCTIONS</th>
                <th className="px-4 py-3">LATEST REVISION</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-100 dark:divide-zinc-800/60">
              {filteredItems.map((c) => (
                <tr key={c.name} className="transition hover:bg-zinc-50/60 dark:hover:bg-zinc-800/30">
                  <td className="px-4 py-3 font-medium text-zinc-900 dark:text-zinc-100">{c.name}</td>
                  <td className="px-4 py-3 font-mono text-[11px] text-zinc-600 dark:text-zinc-400">
                    <Link
                      href={`/infrastructure?view=composites&kind=${encodeURIComponent(c.compositeType)}`}
                      className="inline-flex items-center gap-1 font-semibold text-cyan-600 hover:text-cyan-500 hover:underline dark:text-cyan-400"
                      title={`Filter composites by ${c.compositeType}`}
                    >
                      <span>{c.compositeType}</span>
                      <svg className="h-3 w-3 opacity-60" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                      </svg>
                    </Link>
                  </td>
                  <td className="px-4 py-3">
                    <span className="inline-flex items-center rounded-md bg-emerald-50 px-2 py-0.5 text-[11px] font-semibold text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400">
                      {c.mode}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-zinc-600 dark:text-zinc-400">{c.steps}</td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-1">
                      {c.functions && c.functions.length > 0 ? (
                        c.functions.map((fn) => (
                          <span
                            key={fn}
                            className="inline-flex items-center rounded bg-zinc-100 px-1.5 py-0.5 font-mono text-[10px] text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300"
                          >
                            {fn}
                          </span>
                        ))
                      ) : (
                        <span className="text-zinc-400">—</span>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3 font-mono text-[11px] text-zinc-500 dark:text-zinc-400">
                    {c.latestRevision ?? "—"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// MRDs & MRAPs
// ---------------------------------------------------------------------------

function MRDsSection() {
  const [search, setSearch] = useState("");
  const q = useQuery({ queryKey: ["platform", "mrds"], queryFn: api.platformMRDs });

  if (q.isPending) return <Loading />;
  if (q.isError) return <ErrorState error={q.error} />;
  if (!q.data.available) {
    return (
      <div className="rounded-xl border border-zinc-200 bg-zinc-50/50 p-6 text-center text-xs text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/30">
        API indisponível: este cluster não roda Crossplane v2 (ManagedResourceDefinitions).
      </div>
    );
  }

  const items = (q.data.items ?? []).filter(
    (m) =>
      !search ||
      m.name.toLowerCase().includes(search.toLowerCase()) ||
      m.kind.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <input
          type="text"
          aria-label="Search MRDs"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search MRDs..."
          className="h-8 w-56 rounded-md border border-zinc-200 bg-white px-3 text-xs text-zinc-900 shadow-sm transition placeholder:text-zinc-400 focus:border-zinc-400 focus:outline-none dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100"
        />
        <span className="text-xs font-medium text-zinc-500 dark:text-zinc-400">{items.length} results</span>
      </div>

      <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white shadow-xs dark:border-zinc-800 dark:bg-zinc-900">
        <table className="w-full text-left text-xs">
          <thead className="border-b border-zinc-200 bg-zinc-50/80 text-[11px] font-semibold uppercase tracking-wider text-zinc-500 dark:border-zinc-800 dark:bg-zinc-800/40 dark:text-zinc-400">
            <tr>
              <th className="px-4 py-3">MRD</th>
              <th className="px-4 py-3">STATUS</th>
              <th className="px-4 py-3">KIND</th>
              <th className="px-4 py-3">GROUP</th>
              <th className="px-4 py-3">SCOPE</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-100 dark:divide-zinc-800/60">
            {items.map((m) => (
              <tr key={m.name} className="transition hover:bg-zinc-50/60 dark:hover:bg-zinc-800/30">
                <td className="px-4 py-3 font-mono text-[11px] font-medium text-zinc-900 dark:text-zinc-100">{m.name}</td>
                <td className="px-4 py-3">
                  <span
                    className={`inline-flex items-center rounded-md px-2 py-0.5 text-[11px] font-semibold ${
                      m.active
                        ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400"
                        : "bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400"
                    }`}
                  >
                    {m.active ? "Active" : "Inactive"}
                  </span>
                </td>
                <td className="px-4 py-3 font-medium text-zinc-800 dark:text-zinc-200">
                  <Link
                    href={`/infrastructure?view=managed&kind=${encodeURIComponent(m.kind)}`}
                    className="inline-flex items-center gap-1 font-semibold text-cyan-600 hover:text-cyan-500 hover:underline dark:text-cyan-400"
                    title={`Filter managed resources by ${m.kind}`}
                  >
                    <span>{m.kind}</span>
                    <svg className="h-3 w-3 opacity-60" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                    </svg>
                  </Link>
                </td>
                <td className="px-4 py-3 font-mono text-[11px] text-zinc-600 dark:text-zinc-400">{m.group}</td>
                <td className="px-4 py-3 text-zinc-600 dark:text-zinc-400">{m.scope ?? "—"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function MRAPsSection() {
  const q = useQuery({ queryKey: ["platform", "mraps"], queryFn: api.platformMRAPs });

  if (q.isPending) return <Loading />;
  if (q.isError) return <ErrorState error={q.error} />;
  if (!q.data.available) {
    return (
      <div className="rounded-xl border border-zinc-200 bg-zinc-50/50 p-6 text-center text-xs text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/30">
        API indisponível: este cluster não roda Crossplane v2 (ManagedResourceActivationPolicies).
      </div>
    );
  }

  const items = q.data.items ?? [];

  return (
    <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white shadow-xs dark:border-zinc-800 dark:bg-zinc-900">
      <table className="w-full text-left text-xs">
        <thead className="border-b border-zinc-200 bg-zinc-50/80 text-[11px] font-semibold uppercase tracking-wider text-zinc-500 dark:border-zinc-800 dark:bg-zinc-800/40 dark:text-zinc-400">
          <tr>
            <th className="px-4 py-3">POLICY</th>
            <th className="px-4 py-3">STATUS</th>
            <th className="px-4 py-3">ACTIVATES</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-100 dark:divide-zinc-800/60">
          {items.map((p) => {
            const st = normalizeConditions(p.conditions);
            return (
              <tr key={p.name} className="transition hover:bg-zinc-50/60 dark:hover:bg-zinc-800/30">
                <td className="px-4 py-3 font-medium text-zinc-900 dark:text-zinc-100">{p.name}</td>
                <td className="px-4 py-3">
                  <Badge tone={st.tone}>{st.label}</Badge>
                </td>
                <td className="px-4 py-3 font-mono text-[11px] text-zinc-600 dark:text-zinc-400">
                  {p.activate.join(", ") || "—"}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
