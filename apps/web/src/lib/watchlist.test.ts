import { describe, expect, it } from "vitest";
import { parseWatchlist, toggleWatch } from "./watchlist";

const a = { id: "1", name: "pg", kind: "PostgreSQLInstance" };
const b = { id: "2", name: "vm", kind: "VirtualMachine" };

describe("watchlist", () => {
  it("toggles and keeps the newest first", () => {
    expect(toggleWatch([], a)).toEqual([a]);
    expect(toggleWatch([a], b)).toEqual([b, a]);
    expect(toggleWatch([b, a], a)).toEqual([b]);
  });

  it("caps the list at 20 entries", () => {
    const many = Array.from({ length: 20 }, (_, i) => ({ id: `x${i}`, name: `n${i}`, kind: "K" }));
    const out = toggleWatch(many, a);
    expect(out).toHaveLength(20);
    expect(out[0]).toEqual(a);
  });

  it("parses defensively", () => {
    expect(parseWatchlist(null)).toEqual([]);
    expect(parseWatchlist("not json")).toEqual([]);
    expect(parseWatchlist('{"id":"1"}')).toEqual([]);
    expect(parseWatchlist(JSON.stringify([a, { id: 3 }, null, b]))).toEqual([a, b]);
  });
});
