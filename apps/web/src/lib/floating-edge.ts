// Floating edges: connect two nodes at the point where the line between their centers crosses each
// node's border, so edges stay straight and readable when an XR fans out to many MRs.
// Idea from the React Flow "floating edges" example, also used by crossplane-contrib/crossview.

export interface Rect {
  x: number; // top-left
  y: number;
  width: number;
  height: number;
}

export type Side = "top" | "right" | "bottom" | "left";

/** Point where the segment from a's center to b's center leaves rect a. */
export function borderPoint(a: Rect, b: Rect): { x: number; y: number; side: Side } {
  const ax = a.x + a.width / 2;
  const ay = a.y + a.height / 2;
  const bx = b.x + b.width / 2;
  const by = b.y + b.height / 2;
  const dx = bx - ax;
  const dy = by - ay;
  if (dx === 0 && dy === 0) return { x: ax, y: a.y + a.height, side: "bottom" };
  const hw = a.width / 2;
  const hh = a.height / 2;
  // scale the direction vector until it touches the nearest side
  const t = Math.min(dx !== 0 ? hw / Math.abs(dx) : Infinity, dy !== 0 ? hh / Math.abs(dy) : Infinity);
  const x = ax + dx * t;
  const y = ay + dy * t;
  const side: Side = Math.abs(Math.abs(x - ax) - hw) < 0.5 ? (dx > 0 ? "right" : "left") : dy > 0 ? "bottom" : "top";
  return { x, y, side };
}

export function edgeEndpoints(source: Rect, target: Rect) {
  const s = borderPoint(source, target);
  const t = borderPoint(target, source);
  return { sx: s.x, sy: s.y, sourceSide: s.side, tx: t.x, ty: t.y, targetSide: t.side };
}
