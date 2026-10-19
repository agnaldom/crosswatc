import type { NextRequest } from "next/server";

// Server-side proxy to crosswatc-api so the browser never needs the API origin or credentials.
// In local development CROSSWATC_DEV_USER / CROSSWATC_DEV_SQUADS inject dev identity headers;
// the API rejects dev auth in production (spec §49).
const API_URL = process.env.CROSSWATC_API_URL ?? "http://localhost:8080";

// Never cache or pre-render: /api/v1/stream is a long-lived SSE response (spec §45).
export const dynamic = "force-dynamic";

type Ctx = { params: Promise<{ path: string[] }> };

export function GET(req: NextRequest, ctx: Ctx) {
  return proxy(req, ctx);
}

export function POST(req: NextRequest, ctx: Ctx) {
  return proxy(req, ctx);
}

async function proxy(req: NextRequest, ctx: Ctx) {
  const { path } = await ctx.params;
  const target = `${API_URL}/api/v1/${path.map(encodeURIComponent).join("/")}${req.nextUrl.search}`;

  const isStream = req.method === "GET" && path.length === 1 && path[0] === "stream";
  const headers = new Headers({ Accept: isStream ? "text/event-stream" : "application/json" });
  for (const h of ["authorization", "x-correlation-id", "last-event-id", "content-type"]) {
    const v = req.headers.get(h);
    if (v) headers.set(h, v);
  }
  // oauth2-proxy / ingress auth: forward the access token as a bearer when no Authorization is present.
  const forwarded = req.headers.get("x-forwarded-access-token");
  if (forwarded && !headers.has("authorization")) headers.set("authorization", `Bearer ${forwarded}`);
  if (process.env.CROSSWATC_DEV_USER) {
    headers.set("X-Dev-User", process.env.CROSSWATC_DEV_USER);
    headers.set("X-Dev-Squads", process.env.CROSSWATC_DEV_SQUADS ?? "");
    if (process.env.CROSSWATC_DEV_SRE === "true") headers.set("X-Dev-SRE", "true");
  }

  let upstream: Response;
  try {
    // req.signal aborts the upstream request when the browser goes away, closing the SSE connection.
    upstream = await fetch(target, {
      method: req.method,
      headers,
      body: req.method === "GET" ? undefined : await req.text(),
      cache: "no-store",
      signal: req.signal,
    });
  } catch {
    return Response.json(
      { type: "https://crosswatc.io/errors/upstream", title: "API unavailable", status: 502 },
      { status: 502, headers: { "Content-Type": "application/problem+json" } },
    );
  }
  const out = new Headers({ "Content-Type": upstream.headers.get("Content-Type") ?? "application/json" });
  const cid = upstream.headers.get("X-Correlation-Id");
  if (cid) out.set("X-Correlation-Id", cid);
  if (isStream) {
    out.set("Cache-Control", "no-cache, no-transform");
    out.set("X-Accel-Buffering", "no");
  }
  return new Response(upstream.body, { status: upstream.status, headers: out });
}
