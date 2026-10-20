import { describe, expect, it } from "vitest";
import { isQuickActive, quickFilters, toggleQuick } from "./quick-filters";

const [failed, notReady] = quickFilters;

describe("quick filters", () => {
  it("toggles a param keeping the others", () => {
    const p = new URLSearchParams("kind=VirtualMachine&q=pg");
    const on = new URLSearchParams(toggleQuick(p, notReady));
    expect(on.get("ready")).toBe("false");
    expect(on.get("kind")).toBe("VirtualMachine");
    expect(isQuickActive(on, notReady)).toBe(true);
    expect(new URLSearchParams(toggleQuick(on, notReady)).has("ready")).toBe(false);
  });

  it("replaces another value of the same param", () => {
    const p = new URLSearchParams("status=OPERATIONAL");
    expect(isQuickActive(p, failed)).toBe(false);
    expect(toggleQuick(p, failed)).toBe("status=FAILED");
  });
});
