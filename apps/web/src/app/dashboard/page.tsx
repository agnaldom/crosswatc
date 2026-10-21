"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { Badge } from "@/components/Badge";
import { PlatformHealthCard } from "@/components/PlatformHealthCard";
import { Empty, ErrorState, Loading } from "@/components/States";
import { LiveStatus } from "@/components/Timeline";
import { api, getContexts } from "@/lib/api";
import { resourceStateLabel, resourceStateTone, revisionStateTone, toneDot } from "@/lib/labels";
import type { DashboardSummary, ResourceState, RevisionState } from "@/lib/types";

const states = Object.keys(resourceStateLabel) as ResourceState[];
const levelTone = { info: "muted", warning: "warn", error: "error" } as const;

export default function DashboardPage() {
  const q = useQuery({ queryKey: ["resources", "dashboard"], queryFn: api.dashboard });
  const contextsQuery = useQuery({ queryKey: ["contexts"], queryFn: getContexts, staleTime: 30_000 });
  const currentContext = contextsQuery.data?.currentContext || "in-cluster";

  return (
    <div className="mx-auto max-w-6xl space-y-6">
      {/* Top Header & Live Status */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div className="flex flex-wrap items-center gap-3">
            <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-100">
              Infrastructure Overview
            </h1>
            <div className="inline-flex items-center gap-2 rounded-full border border-zinc-200 bg-zinc-50 px-3 py-1 text-xs dark:border-zinc-800 dark:bg-zinc-900/60">
              <span className="relative flex h-2 w-2">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
              </span>
              <span className="font-medium text-zinc-700 dark:text-zinc-300">
                Context: <code className="font-mono">{currentContext}</code>
              </span>
            </div>
          </div>
          <p className="mt-1 text-xs text-zinc-500 dark:text-zinc-400">
            Monitor composed cloud infrastructure, platform control plane, and active claims across your squads.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <LiveStatus />
        </div>
      </div>

      {q.isPending && <Loading />}
      {q.isError && <ErrorState error={q.error} />}
      {q.data && <Summary d={q.data} />}
    </div>
  );
}

function Summary({ d }: { d: DashboardSummary }) {
  return (
    <div className="space-y-6">
      {/* KPI Cards Grid */}
      <section aria-label="Totals by state" className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
        <Link
          href="/infrastructure"
          className="rounded-xl border border-zinc-200 bg-white p-4 shadow-sm transition hover:border-zinc-300 hover:shadow dark:border-zinc-800 dark:bg-[#0e1117] dark:hover:border-zinc-700"
        >
          <div className="flex items-center justify-between">
            <p className="text-[11px] font-bold uppercase tracking-wider text-zinc-400 dark:text-zinc-500">
              Total Resources
            </p>
            <span className="h-2 w-2 rounded-full bg-blue-500" />
          </div>
          <p className="mt-2 text-2xl font-bold tabular-nums text-zinc-900 dark:text-zinc-100">{d.total}</p>
          <p className="mt-1 text-[11px] text-zinc-500 dark:text-zinc-400">Tracked across squads</p>
        </Link>

        {states
          .filter((s) => d.byState[s] !== undefined && d.byState[s]! > 0)
          .slice(0, 5)
          .map((s) => (
            <div
              key={s}
              className="rounded-xl border border-zinc-200 bg-white p-4 shadow-sm dark:border-zinc-800 dark:bg-[#0e1117]"
            >
              <div className="flex items-center justify-between">
                <p className="flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-wider text-zinc-400 dark:text-zinc-500">
                  <span className={`h-2 w-2 rounded-full ${toneDot[resourceStateTone[s]]}`} aria-hidden />
                  {resourceStateLabel[s]}
                </p>
              </div>
              <p className="mt-2 text-2xl font-bold tabular-nums text-zinc-900 dark:text-zinc-100">{d.byState[s]}</p>
              <p className="mt-1 text-[11px] text-zinc-500 dark:text-zinc-400">
                {Math.round(((d.byState[s] ?? 0) / (d.total || 1)) * 100)}% of total
              </p>
            </div>
          ))}
      </section>

      {/* Health & Distributions */}
      <div className="grid gap-6 lg:grid-cols-3">
        <HealthCard health={d.health} totalResources={d.total} />
        <Distribution title="Cloud Providers" counts={d.byProvider} param="provider" />
        <Distribution title="Environments" counts={d.byEnvironment} param="environment" />
      </div>

      {/* Platform Control Plane Card (SRE) */}
      <PlatformHealthCard />

      {/* Two-Column: Needs Attention & In-Progress Revisions */}
      <div className="grid gap-6 lg:grid-cols-2">
        <Card title="Needs Attention" count={d.attention.length}>
          {d.attention.length === 0 ? (
            <Empty>Nothing degraded, failing, or drifting.</Empty>
          ) : (
            <ul className="divide-y divide-zinc-100 dark:divide-zinc-800/80">
              {d.attention.map((a) => (
                <li
                  key={a.id}
                  className="flex items-center justify-between gap-3 px-4 py-3 text-xs transition hover:bg-zinc-50/60 dark:hover:bg-zinc-800/30"
                >
                  <span className="min-w-0">
                    <Link
                      href={`/infrastructure/${a.id}`}
                      className="block truncate font-semibold text-zinc-900 hover:text-blue-600 dark:text-zinc-100 dark:hover:text-blue-400"
                    >
                      {a.name}
                    </Link>
                    <span className="text-[11px] text-zinc-400 dark:text-zinc-500">{a.kind}</span>
                  </span>
                  <span className="flex shrink-0 items-center gap-1.5">
                    {a.drift && <Badge tone="warn">Drift</Badge>}
                    {a.ready === false && a.resourceState === "OPERATIONAL" && <Badge tone="error">Not Ready</Badge>}
                    <Badge tone={resourceStateTone[a.resourceState]}>{resourceStateLabel[a.resourceState]}</Badge>
                  </span>
                </li>
              ))}
            </ul>
          )}
        </Card>

        <Card title="In-Progress Provisioning Revisions" count={d.openRevisions.length}>
          {d.openRevisions.length === 0 ? (
            <Empty>No changes currently in progress.</Empty>
          ) : (
            <ul className="divide-y divide-zinc-100 dark:divide-zinc-800/80">
              {d.openRevisions.map((r) => (
                <li
                  key={`${r.resourceId}-${r.rev}`}
                  className="space-y-2 px-4 py-3 text-xs transition hover:bg-zinc-50/60 dark:hover:bg-zinc-800/30"
                >
                  <div className="flex items-center justify-between gap-3">
                    <span className="min-w-0">
                      <Link
                        href={`/infrastructure/${r.resourceId}`}
                        className="block truncate font-semibold text-zinc-900 hover:text-blue-600 dark:text-zinc-100 dark:hover:text-blue-400"
                      >
                        {r.resourceName}
                      </Link>
                      <span className="text-[11px] text-zinc-400 dark:text-zinc-500">
                        Revision #{r.rev} · {r.kind}
                      </span>
                    </span>
                    <span className="flex shrink-0 items-center gap-2">
                      {r.prUrl && (
                        <a
                          href={r.prUrl}
                          target="_blank"
                          rel="noreferrer"
                          className="rounded border border-zinc-200 px-1.5 py-0.5 text-[10px] font-semibold text-zinc-700 hover:bg-zinc-100 dark:border-zinc-700 dark:text-zinc-300 dark:hover:bg-zinc-800"
                        >
                          PR ↗
                        </a>
                      )}
                      <Badge tone={revisionStateTone(r.state as RevisionState)}>{r.state}</Badge>
                    </span>
                  </div>

                  {/* Provisioning Pipeline Stepper */}
                  <div className="flex items-center gap-1 pt-1">
                    {["CREATED", "VALIDATING", "WAITING_APPROVAL", "APPLYING", "SUCCEEDED"].map((step, idx) => {
                      const isCurrent = r.state === step;
                      const isPast =
                        ["CREATED", "VALIDATING", "WAITING_APPROVAL", "APPLYING", "SUCCEEDED"].indexOf(r.state) >= idx;
                      return (
                        <div key={step} className="flex flex-1 items-center gap-1">
                          <div
                            className={`h-1.5 w-full rounded-full ${
                              isCurrent
                                ? "bg-blue-600 animate-pulse"
                                : isPast
                                ? "bg-emerald-500"
                                : "bg-zinc-200 dark:bg-zinc-800"
                            }`}
                            title={step}
                          />
                        </div>
                      );
                    })}
                  </div>
                </li>
              ))}
            </ul>
          )}
        </Card>
      </div>

      {/* Recent Activity Events */}
      <Card title="Recent Events & Activity" count={d.recentEvents.length}>
        {d.recentEvents.length === 0 ? (
          <Empty>No recent events recorded.</Empty>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="border-b border-zinc-200 bg-zinc-50/75 text-[10px] font-semibold uppercase tracking-wider text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/40 dark:text-zinc-400">
                <tr>
                  <th className="px-4 py-2.5">Time</th>
                  <th className="px-4 py-2.5">Resource</th>
                  <th className="px-4 py-2.5">Level</th>
                  <th className="px-4 py-2.5">Event Details</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-100 dark:divide-zinc-800/80">
                {d.recentEvents.map((e) => (
                  <tr key={e.eventId} className="transition hover:bg-zinc-50/50 dark:hover:bg-zinc-800/20">
                    <td className="whitespace-nowrap px-4 py-2.5 text-[11px] text-zinc-400 tabular-nums">
                      {new Date(e.timestamp).toLocaleTimeString()}
                    </td>
                    <td className="whitespace-nowrap px-4 py-2.5 font-medium text-zinc-900 dark:text-zinc-100">
                      {e.resourceId ? (
                        <Link
                          href={`/infrastructure/${e.resourceId}`}
                          className="hover:text-blue-600 hover:underline dark:hover:text-blue-400"
                        >
                          {e.resourceName}
                        </Link>
                      ) : (
                        e.resourceName
                      )}
                    </td>
                    <td className="whitespace-nowrap px-4 py-2.5">
                      <span className="flex items-center gap-1.5">
                        <span
                          className={`h-2 w-2 rounded-full ${toneDot[levelTone[e.level] ?? "muted"]}`}
                          aria-label={e.level}
                        />
                        <span className="text-[10px] uppercase font-bold text-zinc-500 dark:text-zinc-400">{e.level}</span>
                      </span>
                    </td>
                    <td className="px-4 py-2.5 text-zinc-600 dark:text-zinc-300">
                      <p className="line-clamp-1" title={e.message || e.type}>
                        {e.message || e.type}
                      </p>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  );
}

const healthParts = [
  { key: "ready", label: "Ready", bar: "bg-emerald-500", text: "text-emerald-600 dark:text-emerald-400", filter: "true" },
  { key: "notReady", label: "Not Ready", bar: "bg-rose-500", text: "text-rose-600 dark:text-rose-400", filter: "false" },
  { key: "unknown", label: "Unknown", bar: "bg-zinc-400 dark:bg-zinc-600", text: "text-zinc-500 dark:text-zinc-400", filter: "unknown" },
] as const;

function HealthCard({ health, totalResources }: { health: DashboardSummary["health"]; totalResources: number }) {
  const total = health.ready + health.notReady + health.unknown;
  const readyPct = total > 0 ? Math.round((health.ready / total) * 100) : 0;

  return (
    <section
      className="flex flex-col justify-between rounded-xl border border-zinc-200 bg-white p-5 shadow-sm dark:border-zinc-800 dark:bg-[#0e1117]"
      aria-label="Resource health"
    >
      <div>
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">Resource Health</h2>
          <span className="rounded-full bg-emerald-50 px-2 py-0.5 text-[10px] font-bold text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-300">
            {readyPct}% Ready
          </span>
        </div>

        {/* Multi-segment progress bar */}
        <div
          className="mt-4 flex h-3 overflow-hidden rounded-full bg-zinc-100 dark:bg-zinc-800"
          role="img"
          aria-label={`${health.ready} ready, ${health.notReady} not ready, ${health.unknown} unknown`}
        >
          {total > 0 &&
            healthParts.map((p) => (
              <div
                key={p.key}
                className={`${p.bar} transition-all duration-500`}
                style={{ width: `${(health[p.key] / total) * 100}%` }}
              />
            ))}
        </div>
      </div>

      <ul className="mt-5 space-y-2 text-xs">
        {healthParts.map((p) => (
          <li key={p.key}>
            <Link
              href={`/infrastructure?ready=${p.filter}`}
              className="flex items-center justify-between rounded-lg p-1.5 transition hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
            >
              <span className="flex items-center gap-2">
                <span className={`h-2.5 w-2.5 rounded-full ${p.bar}`} aria-hidden />
                <span className="font-medium text-zinc-700 dark:text-zinc-300">{p.label}</span>
              </span>
              <div className="flex items-center gap-2">
                <span className="tabular-nums font-semibold text-zinc-900 dark:text-zinc-100">{health[p.key]}</span>
                <span className="text-[10px] text-zinc-400">
                  ({total > 0 ? Math.round((health[p.key] / total) * 100) : 0}%)
                </span>
              </div>
            </Link>
          </li>
        ))}
      </ul>
    </section>
  );
}

function Distribution({
  title,
  counts,
  param,
}: {
  title: string;
  counts: Record<string, number>;
  param: "provider" | "environment";
}) {
  const entries = Object.entries(counts).sort((a, b) => b[1] - a[1]);
  const total = entries.reduce((acc, [, n]) => acc + n, 0);

  return (
    <section className="flex flex-col justify-between rounded-xl border border-zinc-200 bg-white p-5 shadow-sm dark:border-zinc-800 dark:bg-[#0e1117]">
      <div>
        <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">{title}</h2>
        {entries.length === 0 ? (
          <p className="mt-4 text-xs text-zinc-500 dark:text-zinc-400">No resources categorized yet.</p>
        ) : (
          <ul className="mt-4 space-y-3 text-xs">
            {entries.map(([k, n]) => {
              const pct = total > 0 ? Math.round((n / total) * 100) : 0;
              return (
                <li key={k}>
                  <Link
                    href={`/infrastructure?${param}=${encodeURIComponent(k)}`}
                    className="group block rounded-lg transition hover:bg-zinc-50/80 dark:hover:bg-zinc-800/40"
                  >
                    <div className="flex justify-between font-medium">
                      <span className="capitalize text-zinc-700 group-hover:text-blue-600 dark:text-zinc-300 dark:group-hover:text-blue-400">
                        {k}
                      </span>
                      <div className="flex items-center gap-1.5 tabular-nums">
                        <span className="font-semibold text-zinc-900 dark:text-zinc-100">{n}</span>
                        <span className="text-[10px] text-zinc-400">({pct}%)</span>
                      </div>
                    </div>
                    <div className="mt-1.5 block h-1.5 overflow-hidden rounded-full bg-zinc-100 dark:bg-zinc-800">
                      <div
                        className="h-full rounded-full bg-blue-500 transition-all duration-500"
                        style={{ width: `${pct}%` }}
                      />
                    </div>
                  </Link>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </section>
  );
}

function Card({ title, count, children }: { title: string; count: number; children: React.ReactNode }) {
  return (
    <section className="overflow-hidden rounded-xl border border-zinc-200 bg-white shadow-sm dark:border-zinc-800 dark:bg-[#0e1117]">
      <div className="flex items-center justify-between border-b border-zinc-100 px-5 py-3 dark:border-zinc-800/80">
        <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">{title}</h2>
        <span className="rounded-full bg-zinc-100 px-2 py-0.5 text-[10px] font-bold text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400">
          {count}
        </span>
      </div>
      {children}
    </section>
  );
}
