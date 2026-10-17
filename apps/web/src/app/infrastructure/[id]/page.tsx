"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { use, useState } from "react";
import { Badge } from "@/components/Badge";
import { Lifecycle } from "@/components/Lifecycle";
import { Empty, ErrorState, Loading } from "@/components/States";
import { TopologyView } from "@/components/TopologyView";
import { api } from "@/lib/api";
import { healthTone, resourceStateLabel, resourceStateTone, revisionStateTone } from "@/lib/labels";
import type { Health, TrackedResource } from "@/lib/types";

const tabs = ["Overview", "Lifecycle", "Topology", "Changes", "Events"] as const;
type Tab = (typeof tabs)[number];

export default function ResourcePage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const [tab, setTab] = useState<Tab>("Overview");
  const resource = useQuery({ queryKey: ["resource", id], queryFn: () => api.resource(id) });
  const state = useQuery({ queryKey: ["state", id], queryFn: () => api.state(id), enabled: resource.isSuccess });

  if (resource.isPending) return <Loading />;
  if (resource.isError) return <ErrorState error={resource.error} />;
  const r = resource.data;

  return (
    <div className="mx-auto max-w-6xl">
      <Link href="/infrastructure" className="text-sm text-zinc-500 hover:underline">← My Infrastructure</Link>
      <header className="mt-3 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{r.name}</h1>
          <div className="mt-2 flex flex-wrap items-center gap-2 text-sm text-zinc-600">
            <Badge tone={resourceStateTone[r.resourceState]}>{resourceStateLabel[r.resourceState]}</Badge>
            <span>{r.kind}</span>·<span className="capitalize">{r.provider}</span>·<span>{r.environment}</span>
          </div>
        </div>
        <div className="flex gap-2">
          <button type="button" disabled title="Disponível na Fase 3"
            className="rounded-md border border-zinc-300 bg-white px-3 py-1.5 text-sm text-zinc-400">Edit Configuration</button>
          {gitopsUrl(r) && (
            <a href={gitopsUrl(r)} target="_blank" rel="noreferrer"
              className="rounded-md border border-zinc-300 bg-white px-3 py-1.5 text-sm hover:bg-zinc-50">Open GitOps</a>
          )}
        </div>
      </header>

      {state.data?.currentRevision && (
        <p className="mt-3 text-sm text-zinc-600">
          Revision {state.data.currentRevision.rev}:{" "}
          <Badge tone={revisionStateTone(state.data.currentRevision.state)}>{state.data.currentRevision.state}</Badge>
          {state.data.currentRevision.prUrl && (
            <a className="ml-2 underline" href={state.data.currentRevision.prUrl} target="_blank" rel="noreferrer">Pull Request</a>
          )}
        </p>
      )}

      <nav className="mt-6 flex gap-1 border-b border-zinc-200" role="tablist">
        {tabs.map((t) => (
          <button key={t} role="tab" aria-selected={tab === t} type="button" onClick={() => setTab(t)}
            className={`-mb-px border-b-2 px-3 py-2 text-sm ${tab === t ? "border-zinc-900 font-medium" : "border-transparent text-zinc-500 hover:text-zinc-800"}`}>
            {t}
          </button>
        ))}
      </nav>

      <section className="mt-5">
        {tab === "Overview" && <Overview resource={r} />}
        {tab === "Lifecycle" && (state.isError ? <ErrorState error={state.error} /> : state.data ? <Lifecycle stages={state.data.stages} /> : <Loading />)}
        {tab === "Topology" && <TopologyTab id={id} />}
        {tab === "Changes" && <Changes id={id} />}
        {tab === "Events" && <Events id={id} />}
      </section>
    </div>
  );
}

function gitopsUrl(r: TrackedResource) {
  if (!r.repository || !r.gitPath) return undefined;
  return `https://github.com/${r.repository}/blob/HEAD/${r.gitPath}`;
}

const healthRows: [keyof Health, string][] = [
  ["crossplane", "Crossplane"],
  ["argocd", "Argo CD"],
  ["cloud", "Cloud"],
  ["configuration", "Configuration"],
  ["secrets", "Secrets"],
];

function Overview({ resource: r }: { resource: TrackedResource }) {
  const health = useQuery({ queryKey: ["health", r.id], queryFn: () => api.health(r.id) });
  return (
    <div className="grid gap-6 md:grid-cols-2">
      <div className="rounded-lg border border-zinc-200 bg-white p-4">
        <h2 className="text-sm font-medium text-zinc-500">Saúde</h2>
        {health.isError && <ErrorState error={health.error} />}
        <dl className="mt-3 space-y-2 text-sm">
          {healthRows.map(([k, label]) => {
            const v = health.data?.[k] ?? "unknown";
            return (
              <div key={k} className="flex items-center justify-between">
                <dt>{label}</dt>
                <dd><Badge tone={healthTone[v]}>{v}</Badge></dd>
              </div>
            );
          })}
        </dl>
      </div>
      <div className="rounded-lg border border-zinc-200 bg-white p-4">
        <h2 className="text-sm font-medium text-zinc-500">Detalhes</h2>
        <dl className="mt-3 grid grid-cols-[8rem_1fr] gap-y-1.5 text-sm">
          <dt className="text-zinc-500">Owner</dt><dd>{r.owner}</dd>
          <dt className="text-zinc-500">Região</dt><dd>{r.region ?? "—"}</dd>
          <dt className="text-zinc-500">Claim</dt><dd className="font-mono text-xs">{r.claimRef.namespace}/{r.claimRef.name}</dd>
          <dt className="text-zinc-500">Composition</dt><dd className="font-mono text-xs">{r.composition?.name ?? "—"}</dd>
          <dt className="text-zinc-500">Repositório</dt><dd className="font-mono text-xs">{r.repository || "—"}</dd>
          <dt className="text-zinc-500">Path</dt><dd className="break-all font-mono text-xs">{r.gitPath || "—"}</dd>
          <dt className="text-zinc-500">Correlation</dt><dd className="break-all font-mono text-xs">{r.correlationId}</dd>
        </dl>
      </div>
    </div>
  );
}

function TopologyTab({ id }: { id: string }) {
  const q = useQuery({ queryKey: ["topology", id], queryFn: () => api.topology(id) });
  if (q.isPending) return <Loading />;
  if (q.isError) return <ErrorState error={q.error} />;
  return <TopologyView topology={q.data} />;
}

function Changes({ id }: { id: string }) {
  const q = useQuery({ queryKey: ["revisions", id], queryFn: () => api.revisions(id) });
  if (q.isPending) return <Loading />;
  if (q.isError) return <ErrorState error={q.error} />;
  if (!q.data.items.length) return <Empty>Nenhuma mudança registrada.</Empty>;
  return (
    <ul className="space-y-3">
      {q.data.items.map((rv) => (
        <li key={rv.id} className="rounded-lg border border-zinc-200 bg-white p-4 text-sm">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-semibold">REV {rv.rev}</span>
            <span className="text-zinc-500">{rv.kind}</span>
            <Badge tone={revisionStateTone(rv.state)}>{rv.state}</Badge>
            {rv.prUrl && <a href={rv.prUrl} target="_blank" rel="noreferrer" className="underline">PR #{rv.prNumber}</a>}
          </div>
          {rv.failureReason && (
            <p className="mt-2 text-rose-700">
              {rv.failureReason.userMessage}
              {rv.failureReason.recommendedAction && <span className="block text-zinc-600">{rv.failureReason.recommendedAction}</span>}
            </p>
          )}
          <p className="mt-1 text-xs text-zinc-500">
            {rv.startedAt && `Início ${new Date(rv.startedAt).toLocaleString("pt-BR")}`}
            {rv.completedAt && ` · Fim ${new Date(rv.completedAt).toLocaleString("pt-BR")}`}
          </p>
        </li>
      ))}
    </ul>
  );
}

const levelTone = { info: "muted", warning: "warn", error: "error" } as const;

function Events({ id }: { id: string }) {
  const q = useQuery({ queryKey: ["events", id], queryFn: () => api.events(id) });
  if (q.isPending) return <Loading />;
  if (q.isError) return <ErrorState error={q.error} />;
  if (!q.data.items.length) return <Empty>Nenhum evento.</Empty>;
  return (
    <ol className="divide-y divide-zinc-100 rounded-lg border border-zinc-200 bg-white text-sm">
      {q.data.items.map((e) => (
        <li key={e.eventId} className="flex gap-4 px-4 py-2.5">
          <time className="w-36 shrink-0 font-mono text-xs text-zinc-500" dateTime={e.timestamp}>
            {new Date(e.timestamp).toLocaleString("pt-BR")}
          </time>
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <Badge tone={levelTone[e.level]}>{e.source}</Badge>
              <span className="font-mono text-xs text-zinc-500">{e.type}</span>
            </div>
            {e.message && <p className="mt-0.5">{e.message}</p>}
          </div>
        </li>
      ))}
    </ol>
  );
}
