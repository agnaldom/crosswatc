"use client";

import { useQuery } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { Badge } from "@/components/Badge";
import { Icon } from "@/components/Icon";
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

/** TopHeader trigger + Ctrl/⌘+K palette searching resources by name prefix (squad-scoped by the API). */
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
      <button
        type="button"
        onClick={() => setOpen(true)}
        aria-label="Search resources"
        className="flex items-center gap-2 px-2.5 py-1 rounded bg-surface-container hover:bg-surface-container-high transition-colors text-on-surface-variant cursor-pointer border border-surface-container-high/40"
      >
        <Icon name="search" className="h-3.5 w-3.5 text-outline" />
        <span className="font-body-sm text-body-sm pr-space-md text-outline hidden sm:inline">
          Search resources, XRDs...
        </span>
        <kbd className="px-1.5 py-0.5 rounded bg-surface-container-highest font-mono text-[10px] text-on-surface-variant">
          ⌘K
        </kbd>
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
    ? (results.data?.items ?? []).map((r) => ({
        id: r.id,
        name: r.name,
        kind: r.kind,
        state: r.resourceState,
      }))
    : watched;

  useEffect(() => input.current?.focus(), []);
  const active = Math.min(sel, Math.max(0, hits.length - 1));

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
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-black/60 backdrop-blur-sm pt-[15vh]"
      onMouseDown={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Search resources"
        onMouseDown={(e) => e.stopPropagation()}
        className="w-full max-w-lg overflow-hidden rounded-xl border border-surface-container-high bg-surface-container-low shadow-2xl"
      >
        <div className="flex items-center gap-2 border-b border-surface-container px-4 py-3">
          <Icon name="search" className="h-4 w-4 text-primary" />
          <input
            ref={input}
            value={term}
            onChange={(e) => {
              setTerm(e.target.value);
              setSel(0);
            }}
            onKeyDown={onKeyDown}
            placeholder="Search resources by name…"
            aria-label="Search resources by name"
            role="combobox"
            aria-expanded={hits.length > 0}
            aria-controls="global-search-results"
            aria-activedescendant={hits[active] ? `search-hit-${hits[active].id}` : undefined}
            className="w-full bg-transparent text-sm text-on-surface placeholder:text-outline outline-none"
          />
        </div>
        <ul id="global-search-results" role="listbox" className="max-h-80 overflow-y-auto py-1 text-sm divide-y divide-surface-container/30">
          {!q && hits.length > 0 && (
            <li className="px-4 py-1.5 text-xs uppercase tracking-wide text-outline font-mono">
              Watching
            </li>
          )}
          {hits.map((h, i) => (
            <li
              key={h.id}
              id={`search-hit-${h.id}`}
              role="option"
              aria-selected={i === active}
              onClick={() => go(h)}
              className={`flex cursor-pointer items-center justify-between px-4 py-2.5 transition ${
                i === active
                  ? "bg-[#2563eb] text-white"
                  : "text-on-surface hover:bg-surface-container"
              }`}
            >
              <div className="min-w-0">
                <span className="font-medium">{h.name}</span>
                <span className={`ml-2 text-xs ${i === active ? "text-white/80" : "text-outline"}`}>{h.kind}</span>
              </div>
              {h.state && (
                <Badge tone={resourceStateTone[h.state]}>
                  {resourceStateLabel[h.state]}
                </Badge>
              )}
            </li>
          ))}
          {q && hits.length === 0 && !results.isPending && (
            <li className="px-4 py-6 text-center text-xs text-outline">
              No resources found matching &quot;{q}&quot;
            </li>
          )}
        </ul>
      </div>
    </div>
  );
}
