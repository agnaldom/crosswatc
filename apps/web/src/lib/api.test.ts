import { afterEach, expect, it, vi } from "vitest";
import { api, ApiError } from "./api";

afterEach(() => vi.unstubAllGlobals());

it("uses backend filter names and encodes history cursors", async () => {
  const fetch = vi.fn().mockImplementation(() => Promise.resolve(Response.json({ items: [], nextCursor: null })));
  vi.stubGlobal("fetch", fetch);
  await api.listResources({ kind: "PostgreSQLInstance", ready: "unknown", limit: 50, cursor: "a+b/=" });
  const url = new URL(fetch.mock.calls[0][0], "http://localhost");
  expect(url.pathname).toBe("/api/v1/resources");
  expect(Object.fromEntries(url.searchParams)).toEqual({ kind: "PostgreSQLInstance", ready: "unknown", limit: "50", cursor: "a+b/=" });
  for (const read of [api.events, api.revisions, api.audit]) {
    await read("resource-1", "a+b/=");
    expect(new URL(fetch.mock.calls.at(-1)![0], "http://localhost").searchParams.get("cursor")).toBe("a+b/=");
  }
});

it("preserves backend error detail and correlation for denied requests", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(Response.json({ title: "Forbidden", detail: "Owning squad only", correlationId: "trace-1" }, { status: 403 })));
  await expect(api.me()).rejects.toMatchObject({ status: 403, message: "Forbidden: Owning squad only", correlationId: "trace-1" } satisfies Partial<ApiError>);
});
