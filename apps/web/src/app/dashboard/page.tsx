"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { Badge } from "@/components/Badge";
import { Empty, ErrorState, Loading } from "@/components/States";
import { LiveStatus } from "@/components/Timeline";
import { api } from "@/lib/api";
import { resourceStateLabel, resourceStateTone, revisionStateTone, toneDot } from "@/lib/labels";
import type { DashboardSummary, ResourceState, RevisionState } from "@/lib/types";

const states = Object.keys(resourceStateLabel) as ResourceState[];
const levelTone = { info: "muted", warning: "warn", error: "error" } as const;

export default function DashboardPage() {
  // Under ["resources", …] so SSE resource/revision updates refresh it (invalidationsFor).
  const q = useQuery({ queryKey: ["resources", "dashboard"], queryFn: api.dashboard });
  return (
    <div className="mx-auto max-w-6xl">
      <div className="flex items-baseline justify-between gap-4">
        <h1 className="text-2xl font-semibold tracking-tight">Dashboard</h1>
        <LiveStatus />
      </div>
      <p className="mt-1 text-sm text-zinc-600">Visão geral dos recursos das suas squads.</p>
      {q.isPending && <Loading />}
      {q.isError && <ErrorState error={q.error} />}
      {q.data && <Summary d={q.data} />}
    </div>
  );
}

function Summary({ d }: { d: DashboardSummary }) {
  return (
    <>
      <section aria-label="Totais por estado" className="mt-6 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
        <Link href="/infrastructure" className="rounded-lg border border-zinc-200 bg-white p-4 hover:border-zinc-300">
          <p className="text-xs font-medium uppercase tracking-wide text-zinc-500">Total</p>
          <p className="mt-1 text-2xl font-semibold tabular-nums">{d.total}</p>
        </Link>
        {states.filter((s) => d.byState[s]).map((s) => (
          <div key={s} className="rounded-lg border border-zinc-200 bg-white p-4">
            <p className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-zinc-500">
              <span className={`h-2 w-2 rounded-full ${toneDot[resourceStateTone[s]]}`} aria-hidden />
              {resourceStateLabel[s]}
            </p>
            <p className="mt-1 text-2xl font-semibold tabular-nums">{d.byState[s]}</p>
          </div>
        ))}
      </section>

      <div className="mt-6 grid gap-6 lg:grid-cols-2">
        <Card title="Precisam de atenção" count={d.attention.length}>
          {d.attention.length === 0 ? <Empty>Nada degradado, com falha ou drift.</Empty> : (
            <ul className="divide-y divide-zinc-100">
              {d.attention.map((a) => (
                <li key={a.id} className="flex items-center justify-between gap-3 px-4 py-2.5 text-sm">
                  <span className="min-w-0">
                    <Link href={`/infrastructure/${a.id}`} className="block truncate font-medium hover:underline">{a.name}</Link>
                    <span className="text-xs text-zinc-500">{a.kind}</span>
                  </span>
                  <span className="flex shrink-0 gap-1.5">
                    {a.drift && <Badge tone="warn">Drift</Badge>}
                    {a.ready === false && a.resourceState === "OPERATIONAL" && <Badge tone="error">Not Ready</Badge>}
                    <Badge tone={resourceStateTone[a.resourceState]}>{resourceStateLabel[a.resourceState]}</Badge>
                  </span>
                </li>
              ))}
            </ul>
          )}
        </Card>

        <Card title="Revisions em andamento" count={d.openRevisions.length}>
          {d.openRevisions.length === 0 ? <Empty>Nenhuma mudança em andamento.</Empty> : (
            <ul className="divide-y divide-zinc-100">
              {d.openRevisions.map((r) => (
                <li key={`${r.resourceId}-${r.rev}`} className="flex items-center justify-between gap-3 px-4 py-2.5 text-sm">
                  <span className="min-w-0">
                    <Link href={`/infrastructure/${r.resourceId}`} className="block truncate font-medium hover:underline">{r.resourceName}</Link>
                    <span className="text-xs text-zinc-500">Revision {r.rev} · {r.kind}</span>
                  </span>
                  <span className="flex shrink-0 items-center gap-2">
                    {r.prUrl && <a href={r.prUrl} target="_blank" rel="noreferrer" className="text-xs underline">PR</a>}
                    <Badge tone={revisionStateTone(r.state as RevisionState)}>{r.state}</Badge>
                  </span>
                </li>
              ))}
            </ul>
          )}
        </Card>
      </div>

      <div className="mt-6">
        <Card title="Eventos recentes" count={d.recentEvents.length}>
          {d.recentEvents.length === 0 ? <Empty>Sem eventos.</Empty> : (
            <ul className="divide-y divide-zinc-100">
              {d.recentEvents.map((e) => (
                <li key={e.eventId} className="flex items-center gap-3 px-4 py-2 text-sm">
                  <span className={`h-2 w-2 shrink-0 rounded-full ${toneDot[levelTone[e.level] ?? "muted"]}`} aria-label={e.level} />
                  <time className="w-36 shrink-0 text-xs tabular-nums text-zinc-500" dateTime={e.timestamp}>
                    {new Date(e.timestamp).toLocaleString("pt-BR")}
                  </time>
                  {e.resourceId ? (
                    <Link href={`/infrastructure/${e.resourceId}`} className="w-44 shrink-0 truncate font-medium hover:underline">{e.resourceName}</Link>
                  ) : <span className="w-44 shrink-0 truncate">{e.resourceName}</span>}
                  <span className="min-w-0 truncate text-zinc-600" title={e.message || e.type}>{e.message || e.type}</span>
                </li>
              ))}
            </ul>
          )}
        </Card>
      </div>
    </>
  );
}

function Card({ title, count, children }: { title: string; count: number; children: React.ReactNode }) {
  return (
    <section className="overflow-hidden rounded-lg border border-zinc-200 bg-white">
      <h2 className="flex items-center gap-2 border-b border-zinc-100 px-4 py-2.5 text-sm font-semibold">
        {title}
        <span className="rounded bg-zinc-100 px-1.5 text-xs font-medium tabular-nums text-zinc-500">{count}</span>
      </h2>
      {children}
    </section>
  );
}
