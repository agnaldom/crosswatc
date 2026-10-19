"use client";

import { Handle, Position, type Node, type NodeProps } from "@xyflow/react";
import { toneClasses, type Tone } from "@/lib/labels";
import type { TopologyNode } from "@/lib/types";

export type ResourceNodeData = {
  node: TopologyNode;
  tone: Tone;
  label: string;
  detail?: string;
  selected?: boolean;
};

export type ResourceFlowNode = Node<ResourceNodeData, "resource">;

export const toneBorder: Record<Tone, string> = {
  ok: "#10b981",
  progress: "#0ea5e9",
  warn: "#f59e0b",
  error: "#f43f5e",
  muted: "#a1a1aa",
};

export function ResourceNode({ data }: NodeProps<ResourceFlowNode>) {
  const { node: n, tone, label, detail, selected } = data;

  // Crossview-aligned Role/Owner label
  let roleTitle = "Managed Resource";
  if (n.type === "claim") {
    roleTitle = "Claim (Owner)";
  } else if (n.type === "composite") {
    roleTitle = selected ? n.kind : "Composite Resource (Owner)";
  } else if (n.type === "composition") {
    roleTitle = "Composition";
  }

  const borderStyle = selected
    ? "border-2 border-cyan-500 shadow-[0_0_16px_rgba(6,182,212,0.4)]"
    : n.type === "composition"
    ? "border border-dashed border-zinc-300 dark:border-zinc-700"
    : "border border-zinc-200 dark:border-zinc-800";

  return (
    <div
      className={`w-[210px] cursor-pointer rounded-xl bg-white p-3 text-left shadow-sm transition hover:shadow-md dark:bg-[#161b22] ${borderStyle}`}
      title={detail ?? label}
    >
      {/* Invisible Handles: floating edges compute their own anchor points */}
      <Handle type="target" position={Position.Top} className="!h-1 !w-1 !border-0 !bg-transparent" />

      {/* Role / Type Header */}
      <div className="text-center">
        <div className="text-[11px] font-semibold text-zinc-500 dark:text-zinc-400">
          {roleTitle}
        </div>
        <div className="mt-0.5 truncate text-xs font-bold text-zinc-900 dark:text-zinc-100">
          {n.name}
        </div>
      </div>

      {/* Status & Subtitle Indicator */}
      <div className="mt-2 flex items-center justify-center gap-1.5 border-t border-zinc-100 pt-1.5 dark:border-zinc-800/80">
        <span className={`rounded-full px-2 py-0.5 text-[9px] font-semibold uppercase tracking-wider ${toneClasses[tone]}`}>
          {label}
        </span>
        {n.namespace && (
          <span className="truncate text-[10px] text-zinc-400">ns:{n.namespace}</span>
        )}
      </div>

      <Handle type="source" position={Position.Bottom} className="!h-1 !w-1 !border-0 !bg-transparent" />
    </div>
  );
}
