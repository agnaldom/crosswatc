"use client";

import { useQuery } from "@tanstack/react-query";
import { ErrorState, Loading } from "@/components/States";
import { YamlViewer } from "@/components/YamlViewer";
import { api } from "@/lib/api";

const typeLabel = { claim: "Claim", composite: "XR", "managed-resource": "MR", composition: "Composition" } as const;

export function ManifestTab({ id, object, onObject }: { id: string; object: string; onObject: (o: string) => void }) {
  const q = useQuery({ queryKey: ["manifest", id, object], queryFn: () => api.manifest(id, object || undefined), refetchInterval: false });
  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-3 text-sm">
        <label htmlFor="manifest-obj" className="text-zinc-600">Object</label>
        <select id="manifest-obj" value={object || q.data?.object.id || ""} onChange={(e) => onObject(e.target.value)}
          className="rounded-md border border-zinc-300 bg-white px-2 py-1 text-sm" disabled={!q.data}>
          {(q.data?.objects ?? []).map((o) => (
            <option key={o.id} value={o.id}>{typeLabel[o.type]} · {o.kind}/{o.name}</option>
          ))}
        </select>
        <span className="text-xs text-zinc-500">Live cluster state (read-only). Sensitive values appear as [REDACTED].</span>
      </div>
      {q.isPending && <Loading />}
      {q.isError && <ErrorState error={q.error} />}
      {q.data && <YamlViewer yaml={q.data.yaml} />}
    </div>
  );
}
