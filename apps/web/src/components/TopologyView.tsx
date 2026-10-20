"use client";

import { Background, Controls, MarkerType, ReactFlow, type Edge } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { useMemo, useState } from "react";
import { normalizeConditions } from "@/lib/conditions";
import type { Tone } from "@/lib/labels";
import { layout } from "@/lib/topology-layout";
import type { Topology, TopologyNode } from "@/lib/types";
import { FloatingEdge } from "./topology/FloatingEdge";
import { Legend } from "./topology/Legend";
import { NodeDetails } from "./topology/NodeDetails";
import { ResourceNode, type ResourceFlowNode } from "./topology/ResourceNode";

const nodeTypes = { resource: ResourceNode };
const edgeTypes = { floating: FloatingEdge };

// Missing/deleting come from the graph; otherwise the Crossplane conditions decide.
export function statusOf(n: TopologyNode): { tone: Tone; label: string; detail?: string } {
  if (n.status === "missing") return { tone: "muted", label: "missing", detail: n.message };
  if (n.status === "deleting") return { tone: "warn", label: "deleting", detail: n.message };
  if (n.status === "none") return { tone: "muted", label: n.message ?? "referência", detail: n.message };
  return n.conditions?.length ? normalizeConditions(n.conditions, n.kind) : { tone: "muted", label: n.status, detail: n.message };
}

export function TopologyView({ topology, nodeActions, nodeExtraActions }: {
  topology: Topology;
  nodeActions?: (n: TopologyNode) => React.ReactNode;
  nodeExtraActions?: (n: TopologyNode) => React.ReactNode;
}) {
  const [selected, setSelected] = useState<string | null>(null);
  const { nodes, edges } = useMemo(() => {
    const pos = new Map(layout(topology).map((p) => [p.id, p]));
    const nodes: ResourceFlowNode[] = topology.nodes.map((n) => {
      const st = statusOf(n);
      return {
        id: n.id, type: "resource", draggable: true,
        position: { x: pos.get(n.id)?.x ?? 0, y: pos.get(n.id)?.y ?? 0 },
        data: { node: n, tone: st.tone, label: st.label, detail: st.detail, selected: n.id === selected },
      };
    });
    const edges: Edge[] = topology.edges.map((e) => ({
      id: `${e.source}->${e.target}`, source: e.source, target: e.target, type: "floating",
      markerEnd: { type: MarkerType.ArrowClosed, width: 12, height: 12, color: "#38bdf8" },
      style: { stroke: "#38bdf8", strokeDasharray: "4 4", strokeWidth: 1.8 },
    }));
    return { nodes, edges };
  }, [topology, selected]);

  if (!topology.nodes.length) return <p className="text-sm text-zinc-500">Topologia ainda não disponível.</p>;
  const node = topology.nodes.find((n) => n.id === selected);
  return (
    <div className="space-y-2">
      <Legend />
      <div className="relative h-[520px] overflow-hidden rounded-lg border border-zinc-200 bg-white" aria-label="Topologia Claim → XR → Managed Resources">
        <ReactFlow nodes={nodes} edges={edges} nodeTypes={nodeTypes} edgeTypes={edgeTypes} fitView
          proOptions={{ hideAttribution: true }} nodesConnectable={false}
          onNodeClick={(_, n) => setSelected(n.id)} onPaneClick={() => setSelected(null)}>
          <Background />
          <Controls showInteractive={false} />
        </ReactFlow>
        {node && <NodeDetails node={node} onClose={() => setSelected(null)} actions={<>{nodeActions?.(node)}{nodeExtraActions?.(node)}</>} />}
      </div>
    </div>
  );
}
