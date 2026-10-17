import type { NextRequest } from "next/server";

// Server-side proxy to crosswatc-api so the browser never needs the API origin or credentials.
// In local development CROSSWATC_DEV_USER / CROSSWATC_DEV_SQUADS inject dev identity headers;
// the API rejects dev auth in production (spec §49).
const API_URL = process.env.CROSSWATC_API_URL ?? "http://localhost:8080";

export async function GET(req: NextRequest, ctx: { params: Promise<{ path: string[] }> }) {
  const { path } = await ctx.params;
  const target = `${API_URL}/api/v1/${path.map(encodeURIComponent).join("/")}${req.nextUrl.search}`;

  const headers = new Headers({ Accept: "application/json" });
  for (const h of ["authorization", "x-correlation-id"]) {
    const v = req.headers.get(h);
    if (v) headers.set(h, v);
  }
  if (process.env.CROSSWATC_DEV_USER) {
    headers.set("X-Dev-User", process.env.CROSSWATC_DEV_USER);
    headers.set("X-Dev-Squads", process.env.CROSSWATC_DEV_SQUADS ?? "");
    if (process.env.CROSSWATC_DEV_SRE === "true") headers.set("X-Dev-SRE", "true");
  }

  let upstream: Response;
  try {
    upstream = await fetch(target, { headers, cache: "no-store" });
  } catch {
    return Response.json(
      { type: "https://crosswatc.io/errors/upstream", title: "API unavailable", status: 502 },
      { status: 502, headers: { "Content-Type": "application/problem+json" } },
    );
  }
  const out = new Headers({ "Content-Type": upstream.headers.get("Content-Type") ?? "application/json" });
  const cid = upstream.headers.get("X-Correlation-Id");
  if (cid) out.set("X-Correlation-Id", cid);
  return new Response(upstream.body, { status: upstream.status, headers: out });
}
