import { describe, expect, it } from "vitest";
import { borderPoint, edgeEndpoints } from "./floating-edge";

const r = (x: number, y: number, width = 200, height = 80) => ({ x, y, width, height });

describe("floating edges", () => {
  it("vertical stack connects bottom → top centers", () => {
    const e = edgeEndpoints(r(0, 0), r(0, 200));
    expect(e).toMatchObject({ sx: 100, sy: 80, sourceSide: "bottom", tx: 100, ty: 200, targetSide: "top" });
  });
  it("side by side connects right → left", () => {
    const e = edgeEndpoints(r(0, 0), r(400, 0));
    expect(e).toMatchObject({ sx: 200, sy: 40, sourceSide: "right", tx: 400, ty: 40, targetSide: "left" });
  });
  it("diagonal leaves through the correct border and stays on it", () => {
    const p = borderPoint(r(0, 0), r(600, 130));
    expect(p.side).toBe("right");
    expect(p.x).toBe(200);
    expect(p.y).toBeGreaterThan(40);
    expect(p.y).toBeLessThanOrEqual(80);
    const q = borderPoint(r(0, 0), r(20, 400));
    expect(q.side).toBe("bottom");
    expect(q.y).toBe(80);
  });
  it("overlapping centers fall back to bottom", () => {
    expect(borderPoint(r(0, 0), r(0, 0))).toMatchObject({ side: "bottom", y: 80 });
  });
});
