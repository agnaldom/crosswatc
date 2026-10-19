import { describe, expect, it } from "vitest";
import { COLUMN_WIDTH, ROW_HEIGHT, layout } from "./topology-layout";
import type { Topology, TopologyNode } from "./types";

const node = (id: string): TopologyNode => ({ id, type: "managed-resource", kind: "K", name: id, status: "ready" });

describe("layout", () => {
  it("places claim → xr → mrs in layers, centring narrower rows", () => {
    const t: Topology = {
      nodes: ["claim", "xr", "a", "b", "c"].map(node),
      edges: [
        { source: "claim", target: "xr" },
        { source: "xr", target: "a" },
        { source: "xr", target: "b" },
        { source: "xr", target: "c" },
      ],
    };
    const pos = Object.fromEntries(layout(t).map((p) => [p.id, p]));
    expect(pos.claim.depth).toBe(0);
    expect(pos.xr.depth).toBe(1);
    expect(pos.a.y).toBe(2 * ROW_HEIGHT);
    expect(pos.claim.x).toBe(COLUMN_WIDTH); // centred over 3 columns
    expect([pos.a.x, pos.b.x, pos.c.x]).toEqual([0, COLUMN_WIDTH, 2 * COLUMN_WIDTH]);
  });

  it("uses the longest path for nodes reachable at several depths", () => {
    const t: Topology = {
      nodes: ["r", "m", "x"].map(node),
      edges: [
        { source: "r", target: "m" },
        { source: "r", target: "x" },
        { source: "x", target: "m" },
      ],
    };
    const pos = Object.fromEntries(layout(t).map((p) => [p.id, p]));
    expect(pos.m.depth).toBe(2);
  });

  it("tolerates dangling edges and empty graphs", () => {
    expect(layout({ nodes: [], edges: [] })).toEqual([]);
    const pos = layout({ nodes: [node("a")], edges: [{ source: "a", target: "ghost" }] });
    expect(pos).toHaveLength(1);
  });
});
