"use client";

import { useQuery } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { Badge } from "@/components/Badge";
import { api } from "@/lib/api";
import { resourceStateLabel, resourceStateTone } from "@/lib/labels";
import { useWatchlist } from "@/lib/watchlist";

interface Hit {
  id: string;
  name: string;
  kind: string;
  state?: keyof typeof resourceStateLabel;
}

function useDebounced<T>(v: T, ms: number) {
  const [d, setD] = useState(v);
  useEffect(() => {
    const t = setTimeout(() => setD(v), ms);
    return () => clearTimeout(t);
  }, [v, ms]);
  return d;
}

/** Sidebar trigger + Ctrl/⌘+K palette searching resources by name prefix (squad-scoped by the API). */
export function GlobalSearch() {
  const [open, setOpen] = useState(false);
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setOpen((o) => !o);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);
  return (
    <>
      <button type="button" onClick={() => setOpen(true)} aria-label="Search resources"
        className="mt-5 flex w-full items-center justify-between rounded-md border border-zinc-200 bg-zinc-50 px-2 py-1.5 text-sm text-zinc-500 hover:bg-zinc-100">
        <span>Search…</span>
        <kbd className="rounded border border-zinc-200 bg-white px-1 text-[10px] font-medium">⌘K</kbd>
      </button>
      {open && <Palette onClose={() => setOpen(false)} />}
    </>
  );
}

function Palette({ onClose }: { onClose: () => void }) {
  const router = useRouter();
  const input = useRef<HTMLInputElement>(null);
  const [term, setTerm] = useState("");
  const [sel, setSel] = useState(0);
  const q = useDebounced(term.trim(), 200);
  const { list: watched } = useWatchlist();
  const results = useQuery({
    queryKey: ["search", q],
    queryFn: () => api.listResources({ q, limit: 8 }),
    enabled: q.length > 0,
    refetchInterval: false,
  });
  const hits: Hit[] = q
    ? (results.data?.items ?? []).map((r) => ({ id: r.id, name: r.name, kind: r.kind, state: r.resourceState }))
    : watched;
  useEffect(() => input.current?.focus(), []);
  const active = Math.min(sel, hits.length - 1); // results may shrink while typing

  const go = (h?: Hit) => {
    if (!h) return;
    onClose();
    router.push(`/infrastructure/${h.id}`);
  };
  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") onClose();
    else if (e.key === "ArrowDown") setSel(Math.min(active + 1, hits.length - 1));
    else if (e.key === "ArrowUp") setSel(Math.max(active - 1, 0));
    else if (e.key === "Enter") go(hits[active]);
    else return;
    e.preventDefault();
  };

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center bg-zinc-900/30 pt-[15vh]" onMouseDown={onClose}>
      <div role="dialog" aria-modal="true" aria-label="Search resources" onMouseDown={(e) => e.stopPropagation()}
        className="w-full max-w-lg overflow-hidden rounded-lg border border-zinc-200 bg-white shadow-xl">
        <input ref={input} value={term} onChange={(e) => {
            setTerm(e.target.value);
            setSel(0);
          }} onKeyDown={onKeyDown}
          placeholder="Search resources by name…" aria-label="Search resources by name" role="combobox"
          aria-expanded={hits.length > 0} aria-controls="global-search-results"
          aria-activedescendant={hits[active] ? `search-hit-${hits[active].id}` : undefined}
          className="w-full border-b border-zinc-200 px-4 py-3 text-sm outline-none" />
        <ul id="global-search-results" role="listbox" className="max-h-80 overflow-y-auto py-1 text-sm">
          {!q && hits.length > 0 && <li className="px-4 py-1 text-xs uppercase tracking-wide text-zinc-400">Watching</li>}
          {hits.map((h, i) => (
            <li key={h.id} id={`search-hit-${h.id}`} role="option" aria-selected={i === active}
              onMouseEnter={() => setSel(i)} onClick={() => go(h)}
              className={`flex cursor-pointer items-center justify-between gap-3 px-4 py-2 ${i === active ? "bg-zinc-100" : ""}`}>
              <span className="min-w-0">
                <span className="block truncate font-medium">{h.name}</span>
                <span className="block text-xs text-zinc-500">{h.kind}</span>
              </span>
              {h.state && <Badge tone={resourceStateTone[h.state]}>{resourceStateLabel[h.state]}</Badge>}
            </li>
          ))}
          {q && results.isSuccess && hits.length === 0 && <li className="px-4 py-3 text-zinc-500">No resources found.</li>}
          {!q && hits.length === 0 && <li className="px-4 py-3 text-zinc-500">Type to search.</li>}
        </ul>
      </div>
    </div>
  );
}
