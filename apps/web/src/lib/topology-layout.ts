import type { Topology } from "./types";

export interface Positioned {
  id: string;
  x: number;
  y: number;
  depth: number;
}

export const COLUMN_WIDTH = 240;
export const ROW_HEIGHT = 130;

/**
 * Layered top-down layout: depth = longest distance from a root (node without incoming edges);
 * siblings are spread horizontally and centred. Cycles are ignored (Crossplane graphs are trees/DAGs).
 */
export function layout(t: Topology): Positioned[] {
  const incoming = new Map<string, number>();
  const children = new Map<string, string[]>();
  for (const n of t.nodes) {
    incoming.set(n.id, 0);
    children.set(n.id, []);
  }
  for (const e of t.edges) {
    if (!incoming.has(e.source) || !incoming.has(e.target)) continue;
    incoming.set(e.target, (incoming.get(e.target) ?? 0) + 1);
    children.get(e.source)!.push(e.target);
  }

  const depth = new Map<string, number>();
  const queue = t.nodes.filter((n) => incoming.get(n.id) === 0).map((n) => n.id);
  for (const id of queue) depth.set(id, 0);
  const visits = new Map<string, number>();
  while (queue.length) {
    const id = queue.shift()!;
    for (const c of children.get(id) ?? []) {
      const d = (depth.get(id) ?? 0) + 1;
      if (d > (depth.get(c) ?? -1)) depth.set(c, d);
      const v = (visits.get(c) ?? 0) + 1;
      visits.set(c, v);
      if (v === incoming.get(c)) queue.push(c);
    }
  }
  for (const n of t.nodes) if (!depth.has(n.id)) depth.set(n.id, 0);

  const rows = new Map<number, string[]>();
  for (const n of t.nodes) {
    const d = depth.get(n.id)!;
    rows.set(d, [...(rows.get(d) ?? []), n.id]);
  }
  const widest = Math.max(1, ...[...rows.values()].map((r) => r.length));
  const out: Positioned[] = [];
  for (const [d, ids] of rows) {
    const offset = ((widest - ids.length) * COLUMN_WIDTH) / 2;
    ids.forEach((id, i) => out.push({ id, depth: d, x: offset + i * COLUMN_WIDTH, y: d * ROW_HEIGHT }));
  }
  return out;
}
