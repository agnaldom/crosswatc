import { describe, test } from "@e2e-dev/web";
import { expect } from "e2e";

describe("Backend API Integration Tests", () => {
  test("responds to system health probes /healthz and /readyz", async () => {
    // Healthz on backend port directly
    const healthz = await fetch("http://127.0.0.1:8088/healthz");
    expect(healthz.status).toBe(200);
    const healthzJson = await healthz.json();
    expect(healthzJson).toEqual({ status: "ok" });

    // Readyz on backend port directly
    const readyz = await fetch("http://127.0.0.1:8088/readyz");
    expect(readyz.status).toBe(200);
    const readyzJson = await readyz.json();
    expect(readyzJson).toEqual({ status: "ok" });
  });

  test("serves authenticated user profile via /api/v1/me", async ({ app }) => {
    const res = await fetch(new URL("/api/v1/me", app.baseUrl));
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(data.user).toBe("admin");
    expect(data.sre).toBe(true);
    expect(data.squads).toContain("sre");
  });

  test("lists resources and filters by query and status", async ({ app }) => {
    // List all
    const allRes = await fetch(new URL("/api/v1/resources", app.baseUrl));
    expect(allRes.status).toBe(200);
    const allData = await allRes.json();
    expect(allData.items.length).toBe(3);

    // Filter by name query
    const qRes = await fetch(new URL("/api/v1/resources?q=redis", app.baseUrl));
    expect(qRes.status).toBe(200);
    const qData = await qRes.json();
    expect(qData.items.length).toBe(1);
    expect(qData.items[0].name).toBe("redis-session-cache");

    // Filter by synced status
    const syncedRes = await fetch(new URL("/api/v1/resources?synced=false", app.baseUrl));
    expect(syncedRes.status).toBe(200);
    const syncedData = await syncedRes.json();
    expect(syncedData.items.length).toBe(1);
    expect(syncedData.items[0].name).toBe("storage-assets-bucket");
  });

  test("retrieves single resource details and returns 404 for non-existent IDs", async ({ app }) => {
    const validId = "b2c1a4e0-9876-4abc-9def-1234567890ab";
    const res = await fetch(new URL(`/api/v1/resources/${validId}`, app.baseUrl));
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(data.name).toBe("db-payments-prod");
    expect(data.kind).toBe("XPostgreSQLInstance");

    // Non-existent ID
    const notFoundRes = await fetch(new URL("/api/v1/resources/non-existent", app.baseUrl));
    expect(notFoundRes.status).toBe(404);
  });

  test("retrieves resource topology, manifest, and kube-events", async ({ app }) => {
    const id = "b2c1a4e0-9876-4abc-9def-1234567890ab";

    // Topology
    const topoRes = await fetch(new URL(`/api/v1/resources/${id}/topology`, app.baseUrl));
    expect(topoRes.status).toBe(200);
    const topoData = await topoRes.json();
    expect(topoData.nodes.length).toBeGreaterThanOrEqual(4);
    expect(topoData.edges.length).toBeGreaterThanOrEqual(3);

    // Manifest
    const manifestRes = await fetch(new URL(`/api/v1/resources/${id}/manifest`, app.baseUrl));
    expect(manifestRes.status).toBe(200);
    const manifestData = await manifestRes.json();
    expect(manifestData.yaml).toContain("apiVersion: database.example.org/v1alpha1");
    expect(manifestData.object.kind).toBe("XPostgreSQLInstance");

    // Kube Events
    const eventsRes = await fetch(new URL(`/api/v1/resources/${id}/kube-events`, app.baseUrl));
    expect(eventsRes.status).toBe(200);
    const eventsData = await eventsRes.json();
    expect(eventsData.items.length).toBeGreaterThanOrEqual(2);
    expect(eventsData.items[0].reason).toBe("ReconcileSuccess");
  });

  test("provides platform catalog endpoints for providers, xrds, and compositions", async ({ app }) => {
    // Providers
    const providersRes = await fetch(new URL("/api/v1/platform/providers", app.baseUrl));
    expect(providersRes.status).toBe(200);
    const providers = await providersRes.json();
    expect(providers.items.length).toBe(2);
    expect(providers.items[0].name).toBe("provider-azure-db");

    // XRDs
    const xrdsRes = await fetch(new URL("/api/v1/platform/xrds", app.baseUrl));
    expect(xrdsRes.status).toBe(200);
    const xrds = await xrdsRes.json();
    expect(xrds.items.length).toBe(1);

    // Compositions
    const compRes = await fetch(new URL("/api/v1/platform/compositions", app.baseUrl));
    expect(compRes.status).toBe(200);
    const comps = await compRes.json();
    expect(comps.items.length).toBe(1);

    // MRDs (Managed Resource Definitions - Crossplane v2)
    const mrdsRes = await fetch(new URL("/api/v1/platform/mrds", app.baseUrl));
    expect(mrdsRes.status).toBe(200);
    const mrds = await mrdsRes.json();
    expect(mrds.available).toBe(true);
    expect(mrds.items.length).toBe(1);
    expect(mrds.items[0].name).toBe("flexibleservers.azure");

    // MRAPs (Managed Resource Activation Policies - Crossplane v2)
    const mrapsRes = await fetch(new URL("/api/v1/platform/mraps", app.baseUrl));
    expect(mrapsRes.status).toBe(200);
    const mraps = await mrapsRes.json();
    expect(mraps.available).toBe(true);
    expect(mraps.items.length).toBe(1);
    expect(mraps.items[0].name).toBe("activate-all-azure");
  });

  test("serves dashboard summary metrics", async ({ app }) => {
    const res = await fetch(new URL("/api/v1/dashboard", app.baseUrl));
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(data.total).toBe(3);
    expect(data.synced).toBe(2);
    expect(data.ready).toBe(1);
    expect(data.byKind.length).toBe(3);
  });
});
