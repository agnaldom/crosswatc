import { Badge } from "@/components/Badge";
import { ConditionCards } from "@/components/ConditionCards";
import { normalizeConditions } from "@/lib/conditions";
import type { TopologyNode } from "@/lib/types";

export function NodeDetails({ node, onClose, actions }: { node: TopologyNode; onClose: () => void; actions?: React.ReactNode }) {
  const st = normalizeConditions(node.conditions, node.kind);
  const rows: [string, string | undefined][] = [
    ["Kind", node.kind],
    ["API", node.apiVersion],
    ["Name", node.name],
    ["Namespace", node.namespace],
    ["External name", node.externalName],
    ["Composition resource", node.compositionResourceName],
    ["Deletion policy", node.deletionPolicy],
    ["Provider", node.provider],
  ];
  return (
    <aside className="absolute inset-y-0 right-0 z-10 w-80 overflow-y-auto border-l border-zinc-200 bg-white p-4 text-sm shadow-lg" aria-label="Resource details">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="text-xs uppercase tracking-wide text-zinc-500">{node.type}</p>
          <h3 className="truncate font-semibold">{node.name}</h3>
        </div>
        <button type="button" onClick={onClose} className="rounded px-2 text-zinc-500 hover:bg-zinc-100" aria-label="Close">×</button>
      </div>
      <div className="mt-2 flex flex-wrap gap-1">
        <Badge tone={node.status === "missing" || node.status === "none" ? "muted" : st.tone}>
          {node.status === "missing" ? "missing" : node.status === "none" ? node.kind : st.label}
        </Badge>
        {st.chips.filter((c) => c.label !== st.label).map((c) => <Badge key={c.label} tone={c.tone}>{c.label}</Badge>)}
      </div>
      {(st.detail || node.message) && <p className="mt-2 rounded bg-zinc-50 p-2 text-xs text-zinc-700">{st.detail ?? node.message}</p>}
      <dl className="mt-3 grid grid-cols-[7.5rem_1fr] gap-y-1 text-xs">
        {rows.filter(([, v]) => v).map(([k, v]) => (
          <div key={k} className="contents"><dt className="text-zinc-500">{k}</dt><dd className="break-all font-mono">{v}</dd></div>
        ))}
      </dl>
      <h4 className="mt-4 text-xs font-semibold uppercase tracking-wide text-zinc-500">Status Conditions</h4>
      <div className="mt-2"><ConditionCards conditions={node.conditions} compact /></div>
      {actions && <div className="mt-4 flex flex-wrap gap-2">{actions}</div>}
    </aside>
  );
}
