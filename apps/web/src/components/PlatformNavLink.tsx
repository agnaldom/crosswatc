"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

/** Shows the Platform entry only for SRE/platform users (GET /api/v1/me). */
export function PlatformNavLink() {
  const [sre, setSre] = useState(false);
  useEffect(() => {
    fetch("/api/v1/me", { cache: "no-store" })
      .then((r) => (r.ok ? r.json() : null))
      .then((me) => setSre(Boolean(me?.sre)))
      .catch(() => setSre(false));
  }, []);
  if (!sre) return null;
  return (
    <>
      <p className="mt-6 text-xs font-medium uppercase tracking-wide text-zinc-500">Platform</p>
      <nav className="mt-2 flex flex-col gap-1 text-sm">
        <Link href="/platform" className="rounded px-2 py-1.5 hover:bg-zinc-100">Crossplane</Link>
      </nav>
    </>
  );
}
