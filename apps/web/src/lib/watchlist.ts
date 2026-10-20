import { useCallback, useSyncExternalStore } from "react";

/** A pinned resource (client-side only, per browser). */
export interface WatchedResource {
  id: string;
  name: string;
  kind: string;
}

export const WATCHLIST_KEY = "crosswatc.watchlist";
const MAX_WATCHED = 20;
const EVENT = "crosswatc:watchlist";

/** Parses the stored watchlist, dropping malformed entries. */
export function parseWatchlist(raw: string | null): WatchedResource[] {
  if (!raw) return [];
  try {
    const v: unknown = JSON.parse(raw);
    if (!Array.isArray(v)) return [];
    return v
      .filter((x): x is WatchedResource => typeof x?.id === "string" && typeof x?.name === "string" && typeof x?.kind === "string")
      .slice(0, MAX_WATCHED);
  } catch {
    return [];
  }
}

/** Adds the resource at the top, or removes it when already watched. */
export function toggleWatch(list: WatchedResource[], r: WatchedResource): WatchedResource[] {
  if (list.some((x) => x.id === r.id)) return list.filter((x) => x.id !== r.id);
  return [{ id: r.id, name: r.name, kind: r.kind }, ...list].slice(0, MAX_WATCHED);
}

let cache: { raw: string | null; list: WatchedResource[] } = { raw: null, list: [] };

function read(): WatchedResource[] {
  const raw = window.localStorage.getItem(WATCHLIST_KEY);
  if (raw !== cache.raw) cache = { raw, list: parseWatchlist(raw) }; // stable snapshot for useSyncExternalStore
  return cache.list;
}

function subscribe(cb: () => void) {
  const onStorage = (e: StorageEvent) => e.key === WATCHLIST_KEY && cb(); // other tabs
  window.addEventListener("storage", onStorage);
  window.addEventListener(EVENT, cb);
  return () => {
    window.removeEventListener("storage", onStorage);
    window.removeEventListener(EVENT, cb);
  };
}

const empty: WatchedResource[] = [];

export function useWatchlist() {
  const list = useSyncExternalStore<WatchedResource[]>(subscribe, read, () => empty);
  const toggle = useCallback((r: WatchedResource) => {
    window.localStorage.setItem(WATCHLIST_KEY, JSON.stringify(toggleWatch(read(), r)));
    window.dispatchEvent(new Event(EVENT));
  }, []);
  const remove = useCallback((id: string) => {
    window.localStorage.setItem(WATCHLIST_KEY, JSON.stringify(read().filter((x) => x.id !== id)));
    window.dispatchEvent(new Event(EVENT));
  }, []);
  const clearAll = useCallback(() => {
    window.localStorage.setItem(WATCHLIST_KEY, JSON.stringify([]));
    window.dispatchEvent(new Event(EVENT));
  }, []);
  return { list, toggle, remove, clearAll, isWatched: (id: string) => list.some((x) => x.id === id) };
}
