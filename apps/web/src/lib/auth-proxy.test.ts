import { afterEach, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
import { GET, POST } from "../app/api/v1/[...path]/route";

afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); });

it("preserves OIDC redirects and multiple cookies without following the identity provider", async () => {
  const headers = new Headers({ Location: "https://idp.example/authorize?state=bound" });
  headers.append("Set-Cookie", "crosswatc_login=random; HttpOnly; Secure; SameSite=Lax");
  headers.append("Set-Cookie", "crosswatc_session=rotated; HttpOnly; Secure; SameSite=Lax");
  const fetch = vi.fn().mockResolvedValue(new Response(null, { status: 302, headers }));
  vi.stubGlobal("fetch", fetch);
  const response = await GET(new NextRequest("https://workspace.example/api/v1/auth/login"), { params: Promise.resolve({ path: ["auth", "login"] }) });
  expect(fetch.mock.calls[0][1].redirect).toBe("manual");
  expect(response.status).toBe(302);
  expect(response.headers.get("Location")).toContain("https://idp.example/authorize");
  expect(response.headers.getSetCookie()).toHaveLength(2);
  expect(response.headers.get("Cache-Control")).toBe("no-store");
});

it("forwards the session and CSRF origin and never injects Dev identity in OIDC mode", async () => {
  vi.stubEnv("CROSSWATC_AUTH_MODE", "oidc");
  vi.stubEnv("CROSSWATC_DEV_USER", "must-not-be-forwarded");
  const fetch = vi.fn().mockResolvedValue(Response.json({ logoutUrl: "https://idp.example/logout" }));
  vi.stubGlobal("fetch", fetch);
  await POST(new NextRequest("https://workspace.example/api/v1/auth/logout", { method: "POST", headers: { Origin: "https://workspace.example", Cookie: "crosswatc_session=opaque" } }), { params: Promise.resolve({ path: ["auth", "logout"] }) });
  const headers = fetch.mock.calls[0][1].headers as Headers;
  expect(headers.get("Origin")).toBe("https://workspace.example");
  expect(headers.get("Cookie")).toBe("crosswatc_session=opaque");
  expect(headers.has("X-Dev-User")).toBe(false);
});
