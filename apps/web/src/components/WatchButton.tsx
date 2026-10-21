"use client";

import { useWatchlist, type WatchedResource } from "@/lib/watchlist";

/** Star toggle that pins a resource to the sidebar watchlist. */
export function WatchButton({ resource, size = "sm" }: { resource: WatchedResource; size?: "sm" | "md" }) {
  const { isWatched, toggle } = useWatchlist();
  const on = isWatched(resource.id);
  const label = on ? `Stop watching ${resource.name}` : `Watch ${resource.name}`;
  return (
    <button type="button" aria-pressed={on} aria-label={label} title={label}
      onClick={(e) => {
        e.preventDefault();
        toggle(resource);
      }}
      className={`rounded leading-none hover:bg-zinc-100 ${size === "md" ? "px-2 py-1.5 text-lg" : "px-1 py-0.5 text-base"} ${on ? "text-amber-500" : "text-zinc-300 hover:text-zinc-500"}`}>
      {on ? "★" : "☆"}
    </button>
  );
}
