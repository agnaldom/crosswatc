import { describe, expect, it, vi, beforeEach } from "vitest";
import {
  listUsers,
  createUser,
  updateUser,
  deleteUser,
  getContexts,
  setCurrentContext,
  checkContextConnection,
  addKubeConfig,
  removeContext,
} from "./api";

describe("settings api", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("listUsers calls /api/v1/users", async () => {
    const mockUsers = [{ id: 1, username: "admin", email: "admin@local", role: "admin", created_at: "now" }];
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify(mockUsers), { status: 200, headers: { "Content-Type": "application/json" } })
    );

    const res = await listUsers();
    expect(res).toEqual(mockUsers);
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/v1/users", expect.objectContaining({ cache: "no-store" }));
  });

  it("createUser calls POST /api/v1/users", async () => {
    const newUser = { id: 2, username: "dev", email: "dev@local", role: "user", created_at: "now" };
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify(newUser), { status: 201, headers: { "Content-Type": "application/json" } })
    );

    const res = await createUser({ username: "dev", email: "dev@local", password: "pwd", role: "user" });
    expect(res).toEqual(newUser);
    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/v1/users",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ username: "dev", email: "dev@local", password: "pwd", role: "user" }),
      })
    );
  });

  it("updateUser calls PUT /api/v1/users/:id", async () => {
    const updated = { id: 2, username: "dev-updated", email: "dev@local", role: "admin", created_at: "now" };
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify(updated), { status: 200, headers: { "Content-Type": "application/json" } })
    );

    const res = await updateUser(2, { username: "dev-updated", role: "admin" });
    expect(res).toEqual(updated);
    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/v1/users/2",
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({ username: "dev-updated", role: "admin" }),
      })
    );
  });

  it("deleteUser calls DELETE /api/v1/users/:id", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify({ success: true }), { status: 200, headers: { "Content-Type": "application/json" } })
    );

    const res = await deleteUser(2);
    expect(res).toEqual({ success: true });
    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/v1/users/2",
      expect.objectContaining({ method: "DELETE" })
    );
  });

  it("getContexts calls GET /api/v1/contexts", async () => {
    const mockContexts = { contexts: ["ctx-1", "ctx-2"], currentContext: "ctx-1", isInCluster: false };
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify(mockContexts), { status: 200, headers: { "Content-Type": "application/json" } })
    );

    const res = await getContexts();
    expect(res).toEqual(mockContexts);
  });

  it("setCurrentContext calls POST /api/v1/contexts/current", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify({ success: true, context: "ctx-2" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );

    const res = await setCurrentContext("ctx-2");
    expect(res).toEqual({ success: true, context: "ctx-2" });
    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/v1/contexts/current",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ context: "ctx-2" }),
      })
    );
  });

  it("checkContextConnection calls GET /api/v1/contexts/connection", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify({ connected: true, context: "ctx-1" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );

    const res = await checkContextConnection("ctx-1");
    expect(res).toEqual({ connected: true, context: "ctx-1" });
    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/v1/contexts/connection?context=ctx-1",
      expect.anything()
    );
  });

  it("addKubeConfig calls POST /api/v1/contexts/add", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify({ success: true, addedContexts: ["new-ctx"], message: "ok" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );

    const res = await addKubeConfig("apiVersion: v1");
    expect(res).toEqual({ success: true, addedContexts: ["new-ctx"], message: "ok" });
  });

  it("removeContext calls DELETE /api/v1/contexts", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify({ success: true, message: "ok" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );

    const res = await removeContext("old-ctx");
    expect(res).toEqual({ success: true, message: "ok" });
  });
});
