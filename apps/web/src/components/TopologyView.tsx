"use client";

import { Background, Controls, ReactFlow, type Edge, type Node } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { useMemo } from "react";
import { normalizeConditions } from "@/lib/conditions";
import type { Tone } from "@/lib/labels";
import { layout } from "@/lib/topology-layout";
import type { NodeStatus, Topology, TopologyNode } from "@/lib/types";

const statusBorder: Record<NodeStatus, string> = {
  ready: "#10b981",
  "not-ready": "#0ea5e9",
  error: "#f43f5e",
  missing: "#a1a1aa",
  deleting: "#f59e0b",
};

const toneBorder: Record<Tone, string> = { ok: "#10b981", progress: "#0ea5e9", warn: "#f59e0b", error: "#f43f5e", muted: "#a1a1aa" };

// Missing/deleting come from the graph itself; otherwise derive from the Crossplane conditions.
function statusOf(n: TopologyNode) {
  if (n.status === "missing") return { tone: "muted" as Tone, label: "missing", detail: n.message };
  if (n.status === "deleting") return { tone: "warn" as Tone, label: "deleting", detail: n.message };
  return n.conditions?.length ? normalizeConditions(n.conditions, n.kind) : { tone: "muted" as Tone, label: n.status, detail: n.message };
}

const typeLabel = { claim: "Claim", composite: "XR", "managed-resource": "MR" } as const;

export function TopologyView({ topology }: { topology: Topology }) {
  const { nodes, edges } = useMemo(() => {
    const pos = new Map(layout(topology).map((p) => [p.id, p]));
    const nodes: Node[] = topology.nodes.map((n) => ({
      id: n.id,
      position: { x: pos.get(n.id)?.x ?? 0, y: pos.get(n.id)?.y ?? 0 },
      data: {
        label: (
          <div className="text-left" title={statusOf(n).detail ?? n.message ?? n.status}>
            <div className="text-[10px] font-semibold uppercase tracking-wide text-zinc-500">
              {typeLabel[n.type]} · {n.kind}
            </div>
            <div className="truncate text-xs font-medium">{n.name}</div>
            <div className="text-[10px] text-zinc-500">
              {statusOf(n).label}
              {n.provider ? ` · ${n.provider}` : ""}
            </div>
          </div>
        ),
      },
      style: { border: `2px solid ${n.status === "missing" || n.status === "deleting" ? statusBorder[n.status] : toneBorder[statusOf(n).tone]}`, borderRadius: 8, width: 200, padding: 8, background: "white" },
      draggable: false,
    }));
    const edges: Edge[] = topology.edges.map((e) => ({ id: `${e.source}->${e.target}`, source: e.source, target: e.target }));
    return { nodes, edges };
  }, [topology]);

  if (!topology.nodes.length) return <p className="text-sm text-zinc-500">Topologia ainda não disponível.</p>;
  return (
    <div className="h-[480px] rounded-lg border border-zinc-200 bg-white" aria-label="Topologia Claim → XR → Managed Resources">
      <ReactFlow nodes={nodes} edges={edges} fitView proOptions={{ hideAttribution: true }} nodesConnectable={false}>
        <Background />
        <Controls showInteractive={false} />
      </ReactFlow>
    </div>
  );
}
