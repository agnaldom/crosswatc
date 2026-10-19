"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useState } from "react";
import { Badge } from "@/components/Badge";
import { Empty, ErrorState, Loading } from "@/components/States";
import { LiveStatus } from "@/components/Timeline";
import { api, type ResourceFilters } from "@/lib/api";
import { resourceStateLabel, resourceStateTone } from "@/lib/labels";
import type { ResourceState } from "@/lib/types";

const states = Object.keys(resourceStateLabel) as ResourceState[];

export default function InfrastructurePage() {
  const [filters, setFilters] = useState<ResourceFilters>({});
  const [cursors, setCursors] = useState<string[]>([]);
  const cursor = cursors.at(-1);
  const q = useQuery({
    queryKey: ["resources", filters, cursor],
    queryFn: () => api.listResources({ ...filters, cursor }),
  });

  const set = (k: keyof ResourceFilters) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    setCursors([]);
    setFilters((f) => ({ ...f, [k]: e.target.value || undefined }));
  };

  return (
    <div className="mx-auto max-w-6xl">
      <div className="flex items-baseline justify-between gap-4">
        <h1 className="text-2xl font-semibold tracking-tight">My Infrastructure</h1>
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
