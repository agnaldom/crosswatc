import { expect, it } from "vitest";
import { bundlePackages } from "./provider-bundles";
import type { CrossplanePackage } from "./types";

it("counts a bundle once per package across registry aliases, versions and digests", () => {
  const refs = ["xpkg.crossplane.io/crossplane-contrib/provider-family-azure", "xpkg.crossplane.io/crossplane-contrib/provider-azure-compute"];
  const packages: CrossplanePackage[] = [
    { kind: "Provider", name: "old-family", package: "xpkg.upbound.io/upbound/provider-family-azure:v1.0.0", conditions: [] },
    { kind: "Provider", name: "new-family", package: `${refs[0]}@sha256:abcd`, conditions: [] },
    { kind: "Provider", name: "other", package: "xpkg.crossplane.io/crossplane-contrib/provider-aws-ec2:v2.0.0", conditions: [] },
  ];
  expect(bundlePackages(refs, packages).map((p) => p.name)).toEqual(["new-family"]);
  expect(bundlePackages(refs, []).length).toBe(0);
  expect(bundlePackages(refs, [...packages, { kind: "Provider", name: "compute", package: `${refs[1]}:v2.8.0`, conditions: [] }]).length).toBe(2);
});

it("shows dependency waits instead of treating added packages as ready", async () => {
  const { providerState } = await import("./provider-bundles");
  const p: CrossplanePackage = { kind: "Provider", name: "azure", package: "repo:v1.0.0", conditions: [{ type: "Installed", status: "True" }, { type: "Healthy", status: "Unknown", message: "cannot resolve package dependencies" }] };
  expect(providerState(p).ready).toBe(false);
  expect(providerState(p).label).toBe("Waiting for dependencies");
  p.conditions![1].status = "False";
  expect(providerState(p).failed).toBe(true);
  p.conditions![1].status = "True";
  expect(providerState(p).ready).toBe(true);
});
