"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { Suspense, useState } from "react";
import { Badge } from "@/components/Badge";
import { Empty, ErrorState, Loading } from "@/components/States";
import { LiveStatus } from "@/components/Timeline";
import { api, type ResourceFilters } from "@/lib/api";
import { resourceStateLabel, resourceStateTone } from "@/lib/labels";
import type { ResourceState } from "@/lib/types";

const states = Object.keys(resourceStateLabel) as ResourceState[];

/** Synced/Ready pill: True, False or unknown (no observed state yet). */
function ConditionPill({ value }: { value?: boolean }) {
  if (value === undefined) return <Badge tone="muted">—</Badge>;
  return <Badge tone={value ? "ok" : "error"}>{value ? "True" : "False"}</Badge>;
}

const conditionOptions = [
  { value: "true", label: "True" },
  { value: "false", label: "False" },
  { value: "unknown", label: "Desconhecido" },
];

export default function InfrastructurePage() {
  return (
    <Suspense fallback={<Loading />}>
      <KindScoped />
    </Suspense>
  );
}

/** The sidebar Kind menu links to ?kind=…; remount the list per kind so paging resets. */
function KindScoped() {
  const kind = useSearchParams().get("kind") ?? undefined;
  return <InfrastructureList key={kind ?? ""} kind={kind} />;
}

function InfrastructureList({ kind }: { kind?: string }) {
  const [filters, setFilters] = useState<ResourceFilters>({});
  const [cursors, setCursors] = useState<string[]>([]);
  const cursor = cursors.at(-1);
  const q = useQuery({
    queryKey: ["resources", kind, filters, cursor],
    queryFn: () => api.listResources({ ...filters, kind, cursor }),
  });

  const set = (k: keyof ResourceFilters) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    setCursors([]);
    setFilters((f) => ({ ...f, [k]: e.target.value || undefined }));
  };

  return (
    <div className="mx-auto max-w-6xl">
      <div className="flex items-baseline justify-between gap-4">
        <h1 className="text-2xl font-semibold tracking-tight">
          My Infrastructure
          {kind && <span className="ml-2 text-lg font-normal text-zinc-500">· {kind}</span>}
        </h1>
        <LiveStatus />
      </div>
      <p className="mt-1 text-sm text-zinc-600">Recursos das suas squads, do Git até a Cloud.</p>

      <div className="mt-6 flex flex-wrap gap-3">
        <input aria-label="Buscar por nome" placeholder="Buscar por nome" onChange={set("q")}
          className="w-64 rounded-md border border-zinc-300 bg-white px-3 py-1.5 text-sm" />
        <input aria-label="Environment" placeholder="Environment" onChange={set("environment")}
          className="w-36 rounded-md border border-zinc-300 bg-white px-3 py-1.5 text-sm" />
        <select aria-label="Provider" onChange={set("provider")} className="rounded-md border border-zinc-300 bg-white px-3 py-1.5 text-sm">
          <option value="">Todos providers</option>
          <option value="azure">Azure</option>
          <option value="aws">AWS</option>
          <option value="huawei">Huawei</option>
        </select>
        <select aria-label="Status" onChange={set("status")} className="rounded-md border border-zinc-300 bg-white px-3 py-1.5 text-sm">
          <option value="">Todos status</option>
          {states.map((s) => (
            <option key={s} value={s}>{resourceStateLabel[s]}</option>
          ))}
        </select>
        {(["synced", "ready"] as const).map((k) => (
          <select key={k} aria-label={k === "synced" ? "Synced" : "Ready"} onChange={set(k)}
            className="rounded-md border border-zinc-300 bg-white px-3 py-1.5 text-sm">
            <option value="">{k === "synced" ? "Synced: todos" : "Ready: todos"}</option>
            {conditionOptions.map((o) => (
              <option key={o.value} value={o.value}>{k === "synced" ? "Synced" : "Ready"}: {o.label}</option>
            ))}
          </select>
        ))}
        {kind && (
          <Link href="/infrastructure" className="self-center text-sm text-zinc-500 hover:underline">Limpar kind</Link>
        )}
      </div>

      <div className="mt-4">
        {q.isPending && <Loading />}
        {q.isError && <ErrorState error={q.error} />}
        {q.data && q.data.items.length === 0 && <Empty>Nenhum recurso encontrado.</Empty>}
        {q.data && q.data.items.length > 0 && (
          <div className="overflow-hidden rounded-lg border border-zinc-200 bg-white">
            <table className="w-full text-left text-sm">
              <thead className="bg-zinc-50 text-xs uppercase tracking-wide text-zinc-500">
                <tr>
                  <th className="px-4 py-2 font-medium">Nome</th>
                  <th className="px-4 py-2 font-medium">Status</th>
                  <th className="px-4 py-2 font-medium">Synced</th>
                  <th className="px-4 py-2 font-medium">Ready</th>
                  <th className="px-4 py-2 font-medium">Tipo</th>
                  <th className="px-4 py-2 font-medium">Provider</th>
                  <th className="px-4 py-2 font-medium">Environment</th>
                  <th className="px-4 py-2 font-medium">Owner</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-100">
                {q.data.items.map((r) => (
                  <tr key={r.id} className="hover:bg-zinc-50">
                    <td className="whitespace-nowrap px-4 py-2.5 font-medium">
                      <Link href={`/infrastructure/${r.id}`} className="hover:underline">{r.name}</Link>
                    </td>
                    <td className="px-4 py-2.5">
                      <Badge tone={resourceStateTone[r.resourceState]}>{resourceStateLabel[r.resourceState]}</Badge>
                    </td>
                    <td className="px-4 py-2.5"><ConditionPill value={r.synced} /></td>
                    <td className="px-4 py-2.5"><ConditionPill value={r.ready} /></td>
                    <td className="px-4 py-2.5 text-zinc-600">{r.kind}</td>
                    <td className="px-4 py-2.5 capitalize text-zinc-600">{r.provider}</td>
                    <td className="px-4 py-2.5 text-zinc-600">{r.environment}</td>
                    <td className="px-4 py-2.5 text-zinc-600">{r.owner}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <div className="mt-3 flex justify-end gap-2 text-sm">
          <button type="button" disabled={!cursors.length} onClick={() => setCursors((c) => c.slice(0, -1))}
            className="rounded-md border border-zinc-300 bg-white px-3 py-1 disabled:opacity-40">Anterior</button>
          <button type="button" disabled={!q.data?.nextCursor} onClick={() => setCursors((c) => [...c, q.data!.nextCursor!])}
            className="rounded-md border border-zinc-300 bg-white px-3 py-1 disabled:opacity-40">Próxima</button>
        </div>
      </div>
    </div>
  );
}
