"use client";

import { useQueries } from "@tanstack/react-query";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { api } from "@/lib/api";
import { resourceStateLabel, resourceStateTone, toneDot } from "@/lib/labels";
import type { TrackedResource } from "@/lib/types";
import { useWatchlist, type WatchedResource } from "@/lib/watchlist";

/** Sidebar "Watching" section: pinned resources with their live state (SSE invalidates ["resource", id]). */
export function WatchlistNav() {
  const { list } = useWatchlist();
  const pathname = usePathname();
  const states = useQueries({
    queries: list.map((w: WatchedResource) => ({ queryKey: ["resource", w.id], queryFn: () => api.resource(w.id) })),
  });
  if (!list.length) return null;
  return (
    <>
      <p className="mt-6 text-xs font-medium uppercase tracking-wide text-zinc-500">Watching</p>
      <nav className="mt-2 flex flex-col gap-0.5 text-sm" aria-label="Watching">
        {list.map((w: WatchedResource, i: number) => {
          const r = states[i]?.data as TrackedResource | undefined;
          const href = `/infrastructure/${w.id}`;
          const tone = r ? resourceStateTone[r.resourceState] : "muted";
          const gone = states[i]?.isError;
          return (
            <Link key={w.id} href={href} aria-current={pathname === href ? "page" : undefined}
              title={`${w.kind} · ${r ? resourceStateLabel[r.resourceState] : gone ? "unavailable" : "loading"}`}
              className={`flex items-center gap-2 rounded px-2 py-1 hover:bg-zinc-100 ${pathname === href ? "bg-zinc-100 font-medium" : "text-zinc-700"} ${gone ? "opacity-50" : ""}`}>
              <span className={`h-2 w-2 shrink-0 rounded-full ${toneDot[tone]}`} aria-hidden />
              <span className="truncate">{w.name}</span>
            </Link>
          );
        })}
      </nav>
    </>
  );
}
