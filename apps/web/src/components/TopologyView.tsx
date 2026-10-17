"use client";

import { Background, Controls, ReactFlow, type Edge, type Node } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { useMemo } from "react";
import { layout } from "@/lib/topology-layout";
import type { NodeStatus, Topology } from "@/lib/types";

const statusBorder: Record<NodeStatus, string> = {
  ready: "#10b981",
  "not-ready": "#0ea5e9",
  error: "#f43f5e",
  missing: "#a1a1aa",
  deleting: "#f59e0b",
};

const typeLabel = { claim: "Claim", composite: "XR", "managed-resource": "MR" } as const;

export function TopologyView({ topology }: { topology: Topology }) {
  const { nodes, edges } = useMemo(() => {
    const pos = new Map(layout(topology).map((p) => [p.id, p]));
    const nodes: Node[] = topology.nodes.map((n) => ({
      id: n.id,
      position: { x: pos.get(n.id)?.x ?? 0, y: pos.get(n.id)?.y ?? 0 },
      data: {
        label: (
          <div className="text-left" title={n.message ?? n.status}>
            <div className="text-[10px] font-semibold uppercase tracking-wide text-zinc-500">
              {typeLabel[n.type]} · {n.kind}
            </div>
            <div className="truncate text-xs font-medium">{n.name}</div>
            <div className="text-[10px] text-zinc-500">
              {n.status}
              {n.provider ? ` · ${n.provider}` : ""}
            </div>
          </div>
        ),
      },
      style: { border: `2px solid ${statusBorder[n.status]}`, borderRadius: 8, width: 200, padding: 8, background: "white" },
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
