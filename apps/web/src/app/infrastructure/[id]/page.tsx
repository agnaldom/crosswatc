"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { use, useState } from "react";
import { Badge } from "@/components/Badge";
import { DriftBanner } from "@/components/DriftBanner";
import { KubeEvents } from "@/components/KubeEvents";
import { Lifecycle } from "@/components/Lifecycle";
import { Empty, ErrorState, Loading } from "@/components/States";
import { LiveStatus, Timeline } from "@/components/Timeline";
import { TopologyView } from "@/components/TopologyView";
import { api } from "@/lib/api";
import { healthTone, resourceStateLabel, resourceStateTone, revisionStateTone } from "@/lib/labels";
import type { Health, Revision, TrackedResource } from "@/lib/types";

const tabs = ["Overview", "Lifecycle", "Topology", "Changes", "Events", "Kubernetes", "Audit"] as const;
type Tab = (typeof tabs)[number];

export default function ResourcePage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const [tab, setTab] = useState<Tab>("Overview");
  const [kubeFilter, setKubeFilter] = useState("");
  const showEvents = (filter: string) => {
    setKubeFilter(filter);
    setTab("Kubernetes");
  };
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
            <span className="ml-2"><LiveStatus /></span>
          </div>
        </div>
        <div className="flex gap-2">
          {r.resourceState === "OPERATIONAL" ? (
            <Link href={`/infrastructure/${r.id}/edit`}
              className="rounded-md bg-zinc-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-zinc-700">Edit Configuration</Link>
          ) : (
            <button type="button" disabled title="Somente recursos operacionais podem ser editados"
              className="rounded-md border border-zinc-300 bg-white px-3 py-1.5 text-sm text-zinc-400">Edit Configuration</button>
          )}
          {gitopsUrl(r) && (
            <a href={gitopsUrl(r)} target="_blank" rel="noreferrer"
              className="rounded-md border border-zinc-300 bg-white px-3 py-1.5 text-sm hover:bg-zinc-50">Open GitOps</a>
          )}
        </div>
      </header>

      {state.data?.drift && <DriftBanner drift={state.data.drift} />}
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
        {tab === "Topology" && <TopologyTab id={id} onEvents={showEvents} />}
        {tab === "Kubernetes" && <KubeEvents id={id} filter={kubeFilter} onFilter={setKubeFilter} />}
        {tab === "Changes" && <Changes id={id} />}
        {tab === "Events" && <Events id={id} />}
        {tab === "Audit" && <Audit id={id} />}
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
      {["OPERATIONAL", "DEGRADED", "FAILED"].includes(r.resourceState) && (
        <div className="rounded-lg border border-rose-200 bg-white p-4 md:col-span-2">
          <h2 className="text-sm font-medium text-rose-800">Zona de perigo</h2>
          <div className="mt-2 flex items-center justify-between gap-4 text-sm">
            <p className="text-zinc-600">Remover este recurso abre um Pull Request que apaga a infraestrutura após aprovação.</p>
            <Link href={`/infrastructure/${r.id}/decommission`}
              className="shrink-0 rounded-md border border-rose-300 px-3 py-1.5 font-medium text-rose-700 hover:bg-rose-50">Decommission</Link>
          </div>
        </div>
      )}
    </div>
  );
}

function TopologyTab({ id, onEvents }: { id: string; onEvents: (filter: string) => void }) {
  const q = useQuery({ queryKey: ["topology", id], queryFn: () => api.topology(id) });
  if (q.isPending) return <Loading />;
  if (q.isError) return <ErrorState error={q.error} />;
  return (
    <TopologyView topology={q.data} nodeActions={(n) => (
      <button type="button" onClick={() => onEvents(`${n.kind}/${n.name}`)}
        className="rounded-md border border-zinc-300 px-2.5 py-1 text-xs hover:bg-zinc-50">Ver eventos do cluster</button>
    )} />
  );
}

function Changes({ id }: { id: string }) {
  const q = useQuery({ queryKey: ["revisions", id], queryFn: () => api.revisions(id) });
  const [open, setOpen] = useState<number | null>(null);
  if (q.isPending) return <Loading />;
  if (q.isError) return <ErrorState error={q.error} />;
  if (!q.data.items.length) return <Empty>Nenhuma mudança registrada.</Empty>;
  return (
    <ul className="space-y-3">
      {q.data.items.map((rv) => (
        <li key={rv.id} className="rounded-lg border border-zinc-200 bg-white text-sm">
          <button type="button" aria-expanded={open === rv.rev} onClick={() => setOpen(open === rv.rev ? null : rv.rev)}
            className="w-full p-4 text-left hover:bg-zinc-50">
            <RevisionSummary rv={rv} />
          </button>
          {["CREATED", "VALIDATING", "WAITING_APPROVAL"].includes(rv.state) && <CancelRevision id={id} rev={rv.rev} />}
          {open === rv.rev && <RevisionHistory id={id} rev={rv.rev} />}
        </li>
      ))}
    </ul>
  );
}

function CancelRevision({ id, rev }: { id: string; rev: number }) {
  const qc = useQueryClient();
  const cancel = useMutation({
    mutationFn: () => api.cancelRevision(id, rev, "Cancelado pelo solicitante"),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["revisions", id] }),
  });
  return (
    <div className="border-t border-zinc-100 px-4 py-2">
      <button type="button" disabled={cancel.isPending}
        onClick={() => window.confirm(`Cancelar a revision ${rev} e fechar o Pull Request?`) && cancel.mutate()}
        className="text-sm text-rose-700 hover:underline disabled:opacity-50">
        {cancel.isPending ? "Cancelando…" : "Cancelar revision"}
      </button>
      {cancel.isError && <ErrorState error={cancel.error} />}
    </div>
  );
}

function RevisionSummary({ rv }: { rv: Revision }) {
  return (
    <>
      <div className="flex flex-wrap items-center gap-2">
        <span className="font-semibold">REV {rv.rev}</span>
        <span className="text-zinc-500">{rv.kind}</span>
        <Badge tone={revisionStateTone(rv.state)}>{rv.state}</Badge>
        {rv.prNumber && <span className="text-zinc-600">PR #{rv.prNumber}</span>}
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
    </>
  );
}

/** Operations and event history of one revision (GET /resources/{id}/revisions/{rev}). */
function RevisionHistory({ id, rev }: { id: string; rev: number }) {
  const q = useQuery({ queryKey: ["revision", id, rev], queryFn: () => api.revision(id, rev) });
  if (q.isPending) return <div className="border-t border-zinc-100 px-4"><Loading /></div>;
  if (q.isError) return <div className="border-t border-zinc-100 p-4"><ErrorState error={q.error} /></div>;
  const d = q.data;
  return (
    <div className="space-y-4 border-t border-zinc-100 p-4">
      {d.prUrl && <a href={d.prUrl} target="_blank" rel="noreferrer" className="underline">Abrir Pull Request #{d.prNumber}</a>}
      {d.operations.map((o) => (
        <p key={o.id} className="text-xs text-zinc-600">
          Operação {o.type} · <Badge tone={o.status === "SUCCEEDED" ? "ok" : o.status === "FAILED" ? "error" : "progress"}>{o.status}</Badge>
          {" "}· solicitada por <span className="font-medium">{o.requestedBy}</span> em {new Date(o.requestedAt).toLocaleString("pt-BR")}
        </p>
      ))}
      {d.events.length ? <Timeline events={[...d.events].reverse()} live={false} /> : <Empty>Nenhum evento nesta revisão.</Empty>}
    </div>
  );
}

function Events({ id }: { id: string }) {
  // Kept fresh by the realtime stream (event.created patches this cache); polls only as a fallback.
  const q = useQuery({ queryKey: ["events", id], queryFn: () => api.events(id) });
  if (q.isPending) return <Loading />;
  if (q.isError) return <ErrorState error={q.error} />;
  if (!q.data.items.length) return <Empty>Nenhum evento.</Empty>;
  return <Timeline events={q.data.items} />;
}

const auditLabel: Record<string, string> = {
  "change.requested": "Mudança solicitada",
  "change.failed": "Falha ao abrir PR",
  "revision.cancelled": "Revision cancelada",
  "revision.superseded": "Revision substituída",
};

function Audit({ id }: { id: string }) {
  const q = useQuery({ queryKey: ["audit", id], queryFn: () => api.audit(id) });
  if (q.isPending) return <Loading />;
  if (q.isError) return <ErrorState error={q.error} />;
  if (!q.data.items.length) return <Empty>Nenhuma ação registrada.</Empty>;
  return (
    <table className="w-full overflow-hidden rounded-lg border border-zinc-200 bg-white text-left text-sm">
      <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
        <tr><th className="px-3 py-2">Quando</th><th className="px-3 py-2">Quem</th><th className="px-3 py-2">O quê</th><th className="px-3 py-2">Antes → Depois</th><th className="px-3 py-2">PR</th></tr>
      </thead>
      <tbody className="divide-y divide-zinc-100">
        {q.data.items.map((a) => (
          <tr key={a.id} className="align-top">
            <td className="whitespace-nowrap px-3 py-2 font-mono text-xs text-zinc-500">{new Date(a.createdAt).toLocaleString("pt-BR")}</td>
            <td className="px-3 py-2">{a.actor}</td>
            <td className="px-3 py-2">{auditLabel[a.action] ?? a.action}</td>
            <td className="px-3 py-2 font-mono text-xs">
              {a.after && Object.entries(a.after).map(([k, v]) => (
                <div key={k}>{k}: {a.before?.[k] !== undefined ? `${String(a.before[k])} → ` : ""}{String(v)}</div>
              ))}
            </td>
            <td className="px-3 py-2">{a.prUrl && <a href={a.prUrl} target="_blank" rel="noreferrer" className="underline">abrir</a>}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
