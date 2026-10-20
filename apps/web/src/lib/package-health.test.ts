import { describe, expect, it } from "vitest";
import { packageHealth } from "./package-health";
import type { CrossplanePackage } from "./types";

const pkg = (name: string, healthy?: string): CrossplanePackage =>
  ({
    kind: "Provider", name, package: `xpkg.upbound.io/${name}:v1`,
    conditions: healthy === undefined ? [] : [{ type: "Installed", status: "True" }, { type: "Healthy", status: healthy }],
  }) as CrossplanePackage;

describe("packageHealth", () => {
  it("splits healthy, unhealthy and pending", () => {
    const h = packageHealth([pkg("provider-azure", "True"), pkg("provider-aws", "False"), pkg("provider-new")]);
    expect(h.total).toBe(3);
    expect(h.healthy).toBe(1);
    expect(h.unhealthy.map((p) => p.name)).toEqual(["provider-aws"]);
    expect(h.pending).toBe(1);
  });
});
