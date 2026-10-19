import { describe, expect, it } from "vitest";
import { normalizeConditions } from "./conditions";

const c = (type: string, status: string, reason?: string, message?: string) => ({ type, status, reason, message });

describe("normalizeConditions", () => {
  it("resources: Synced + Ready", () => {
    expect(normalizeConditions([c("Synced", "True"), c("Ready", "True")], "FlexibleServer")).toMatchObject({ tone: "ok", label: "Ready" });
    const ns = normalizeConditions([c("Synced", "False", "ReconcileError", "AuthorizationFailed"), c("Ready", "False")]);
    expect(ns).toMatchObject({ tone: "error", label: "Not Synced", detail: "ReconcileError: AuthorizationFailed" });
    expect(ns.chips.map((x) => x.label)).toEqual(["Not Synced", "Not Ready"]);
    expect(normalizeConditions([c("Synced", "True"), c("Ready", "False", "Creating")])).toMatchObject({ tone: "progress", label: "Creating" });
    expect(normalizeConditions([c("Synced", "True"), c("Ready", "False", "Unhealthy")])).toMatchObject({ tone: "error", label: "Not Ready" });
    expect(normalizeConditions([c("Synced", "Unknown")])).toMatchObject({ tone: "warn", label: "Pending" });
    expect(normalizeConditions([c("Synced", "True")])).toMatchObject({ tone: "ok", label: "Synced" });
  });

  it("packages: Installed + Healthy", () => {
    expect(normalizeConditions([c("Installed", "True"), c("Healthy", "True")], "Provider")).toMatchObject({ tone: "ok", label: "Healthy" });
    expect(normalizeConditions([c("Installed", "True"), c("Healthy", "False", "UnhealthyPackageRevision")], "Provider")).toMatchObject({
      tone: "error", label: "Unhealthy",
    });
    expect(normalizeConditions([c("Installed", "True")], "Function")).toMatchObject({ tone: "progress", label: "Installed" });
    expect(normalizeConditions([c("Installed", "False")], "Function")).toMatchObject({ tone: "error", label: "Not Installed" });
  });

  it("XRDs: Established + Offered", () => {
    expect(normalizeConditions([c("Established", "True"), c("Offered", "True")], "CompositeResourceDefinition")).toMatchObject({
      tone: "ok", label: "Established & Offered",
    });
    expect(normalizeConditions([c("Established", "True"), c("Offered", "False")], "CompositeResourceDefinition").label).toBe("Not Offered");
    expect(normalizeConditions([c("Established", "True")], "CompositeResourceDefinition").label).toBe("Established");
  });

  it("other kinds and empty", () => {
    expect(normalizeConditions([], "X")).toMatchObject({ tone: "muted", label: "Unknown" });
    expect(normalizeConditions(undefined)).toMatchObject({ label: "Unknown" });
    expect(normalizeConditions([c("Available", "True")]).label).toBe("Active");
    expect(normalizeConditions([c("Available", "False")]).label).toBe("Inactive");
  });
});
