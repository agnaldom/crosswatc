"use client";

import { useQuery } from "@tanstack/react-query";
import { Badge } from "@/components/Badge";
import { Empty, ErrorState, Loading } from "@/components/States";
import { api } from "@/lib/api";

const key = (o: { kind: string; name: string }) => `${o.kind}/${o.name}`;

export function KubeEvents({ id, filter, onFilter }: { id: string; filter: string; onFilter: (f: string) => void }) {
  const q = useQuery({ queryKey: ["kube-events", id], queryFn: () => api.kubeEvents(id) });
  if (q.isPending) return <Loading />;
  if (q.isError) return <ErrorState error={q.error} />;
  const objects = [...new Set(q.data.items.map((e) => key(e.object)))].sort();
  const items = q.data.items.filter((e) => !filter || key(e.object) === filter);
  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-3 text-sm">
        <label htmlFor="obj" className="text-zinc-600">Objeto</label>
        <select id="obj" value={filter} onChange={(e) => onFilter(e.target.value)}
          className="rounded-md border border-zinc-300 bg-white px-2 py-1 text-sm">
          <option value="">Todos (Claim, XR e MRs)</option>
          {objects.map((o) => <option key={o} value={o}>{o}</option>)}
          {filter && !objects.includes(filter) && <option value={filter}>{filter}</option>}
        </select>
        <span className="text-xs text-zinc-500">Eventos do cluster gerados pelos controllers do Crossplane e providers.</span>
      </div>
      {q.data.errors.length > 0 && (
        <p className="rounded-md bg-amber-50 p-2 text-xs text-amber-900">
          Não foi possível ler eventos de: {q.data.errors.map((e) => `${key(e.object)} (${e.error})`).join("; ")}
        </p>
      )}
      {items.length === 0 ? (
        <Empty>Nenhum evento Kubernetes{filter ? ` para ${filter}` : ""}. Eventos expiram no cluster (padrão: 1 h).</Empty>
      ) : (
        <ol className="divide-y divide-zinc-100 rounded-lg border border-zinc-200 bg-white text-sm">
          {items.map((e, i) => (
            <li key={i} className="flex gap-4 px-4 py-2.5">
              <time className="w-36 shrink-0 font-mono text-xs text-zinc-500" dateTime={e.lastSeen}>
                {e.lastSeen ? new Date(e.lastSeen).toLocaleString("pt-BR") : "—"}
              </time>
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <Badge tone={e.type === "Warning" ? "error" : "muted"}>{e.type}</Badge>
                  <span className="font-medium">{e.reason}</span>
                  {e.count > 1 && <span className="text-xs text-zinc-500">×{e.count}</span>}
                  <span className="font-mono text-xs text-zinc-500">{key(e.object)}</span>
                </div>
                <p className="mt-0.5 break-words">{e.message}</p>
                {e.source && <p className="text-xs text-zinc-400">{e.source}</p>}
              </div>
            </li>
          ))}
        </ol>
      )}
    </div>
  );
}
