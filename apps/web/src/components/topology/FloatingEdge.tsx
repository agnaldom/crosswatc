"use client";

import { BaseEdge, getStraightPath, useInternalNode, type EdgeProps } from "@xyflow/react";
import { edgeEndpoints } from "@/lib/floating-edge";

export function FloatingEdge({ id, source, target, markerEnd, style }: EdgeProps) {
  const s = useInternalNode(source);
  const t = useInternalNode(target);
  if (!s || !t) return null;
  const rect = (n: typeof s) => ({
    x: n.internals.positionAbsolute.x,
    y: n.internals.positionAbsolute.y,
    width: n.measured?.width ?? 200,
    height: n.measured?.height ?? 80,
  });
  const { sx, sy, tx, ty } = edgeEndpoints(rect(s), rect(t));
  const [path] = getStraightPath({ sourceX: sx, sourceY: sy, targetX: tx, targetY: ty });
  return <BaseEdge id={id} path={path} markerEnd={markerEnd} style={style} />;
}
