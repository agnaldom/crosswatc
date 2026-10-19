"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { usePathname, useSearchParams } from "next/navigation";
import { api } from "@/lib/api";

/** Sidebar submenu: the caller's resources grouped by Kind (GET /api/v1/resources/kinds). */
export function KindNavLinks() {
  const kinds = useQuery({ queryKey: ["resource-kinds"], queryFn: api.resourceKinds, staleTime: 60_000 });
  const pathname = usePathname();
  const kindParam = useSearchParams().get("kind");
  const active = pathname === "/infrastructure" ? kindParam : null;
  const items = kinds.data?.items ?? [];
  if (!items.length) return null;
  return (
    <>
      <p className="mt-6 text-xs font-medium uppercase tracking-wide text-zinc-500">Resources</p>
      <nav className="mt-2 flex flex-col gap-0.5 text-sm" aria-label="Resources by kind">
        {items.map((k) => (
          <Link key={k.kind} href={`/infrastructure?kind=${encodeURIComponent(k.kind)}`}
            aria-current={active === k.kind ? "page" : undefined}
            className={`flex items-center justify-between gap-2 rounded px-2 py-1 hover:bg-zinc-100 ${active === k.kind ? "bg-zinc-100 font-medium" : "text-zinc-700"}`}>
            <span className="truncate" title={k.kind}>{k.kind}</span>
            <span className="rounded bg-zinc-100 px-1.5 text-xs tabular-nums text-zinc-500">{k.count}</span>
          </Link>
        ))}
      </nav>
    </>
  );
}
