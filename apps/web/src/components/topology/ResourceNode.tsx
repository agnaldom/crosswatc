"use client";

import { Handle, Position, type Node, type NodeProps } from "@xyflow/react";
import { toneClasses, type Tone } from "@/lib/labels";
import type { TopologyNode } from "@/lib/types";

export type ResourceNodeData = { node: TopologyNode; tone: Tone; label: string; detail?: string; selected?: boolean };
export type ResourceFlowNode = Node<ResourceNodeData, "resource">;

export const toneBorder: Record<Tone, string> = { ok: "#10b981", progress: "#0ea5e9", warn: "#f59e0b", error: "#f43f5e", muted: "#a1a1aa" };
const typeLabel = { claim: "Claim", composite: "XR", "managed-resource": "MR" } as const;

export function ResourceNode({ data }: NodeProps<ResourceFlowNode>) {
  const { node: n, tone, label, detail } = data;
  return (
    <div className="w-[220px] cursor-pointer rounded-lg bg-white p-2 text-left shadow-sm transition hover:shadow-md"
      style={{ border: `2px solid ${toneBorder[tone]}`, outline: data.selected ? `3px solid ${toneBorder[tone]}33` : undefined }}
      title={detail ?? label}>
      {/* invisible handles: floating edges compute their own anchor points */}
      <Handle type="target" position={Position.Top} className="!h-1 !w-1 !border-0 !bg-transparent" />
      <div className="flex items-center justify-between gap-1">
        <span className="rounded bg-zinc-900 px-1.5 py-0.5 text-[9px] font-semibold uppercase tracking-wide text-white">
          {typeLabel[n.type]}
        </span>
        <span className="truncate rounded bg-zinc-100 px-1.5 py-0.5 text-[10px] font-medium text-zinc-700">{n.kind}</span>
      </div>
      <div className="mt-1 truncate text-xs font-semibold">{n.name}</div>
      {n.namespace && <div className="truncate text-[10px] text-zinc-500">ns: {n.namespace}</div>}
      <div className="mt-1 flex items-center gap-1">
        <span className={`rounded px-1.5 py-0.5 text-[10px] ring-1 ring-inset ${toneClasses[tone]}`}>{label}</span>
        {n.provider && <span className="text-[10px] text-zinc-500">{n.provider}</span>}
        {n.deletionPolicy === "Orphan" && <span className="text-[10px] text-amber-700" title="deletionPolicy: Orphan">orphan</span>}
      </div>
      <Handle type="source" position={Position.Bottom} className="!h-1 !w-1 !border-0 !bg-transparent" />
    </div>
  );
}
