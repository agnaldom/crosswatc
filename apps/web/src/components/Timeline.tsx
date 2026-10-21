"use client";

import { useSyncExternalStore } from "react";
import { Badge } from "@/components/Badge";
import { useRealtimeStatus } from "@/lib/realtime";
import { describe, groupByDay, isFresh, timeOf } from "@/lib/timeline";
import type { TrackerEvent } from "@/lib/types";

const levelTone = { info: "muted", warning: "warn", error: "error" } as const;
const levelDot = { info: "bg-zinc-400", warning: "bg-amber-500", error: "bg-rose-500" } as const;

// A coarse shared clock so "just arrived" highlights fade without impure reads during render.
let now = Date.now();
const clockListeners = new Set<() => void>();
let clock: ReturnType<typeof setInterval> | undefined;
function subscribeClock(l: () => void) {
  clockListeners.add(l);
  clock ??= setInterval(() => {
    now = Date.now();
    clockListeners.forEach((f) => f());
  }, 5_000);
  return () => {
    clockListeners.delete(l);
    if (!clockListeners.size && clock) {
      clearInterval(clock);
      clock = undefined;
    }
  };
}
const useNow = () => useSyncExternalStore(subscribeClock, () => now, () => 0);

const dayFormat = new Intl.DateTimeFormat("en-US", { weekday: "short", day: "2-digit", month: "short", year: "numeric" });

/** Realtime event timeline (spec §46 Events): newest first, grouped by day, new arrivals highlighted. */
export function Timeline({ events, live = true }: { events: TrackerEvent[]; live?: boolean }) {
  const current = useNow();
  return (
    <div className="space-y-5">
      {groupByDay(events).map((g) => (
        <section key={g.day} aria-label={g.day}>
          <h3 className="mb-2 text-xs font-medium uppercase tracking-wide text-zinc-500">
            {dayFormat.format(new Date(`${g.day}T12:00:00`))}
          </h3>
          <ol className="relative ml-2 border-l border-zinc-200">
            {g.items.map((e) => {
              const fresh = live && isFresh(e.timestamp, current);
              return (
                <li key={e.eventId} className={`relative py-2 pl-5 text-sm transition-colors ${fresh ? "bg-sky-50/70" : ""}`}>
                  <span className={`absolute -left-[5px] top-3.5 h-2.5 w-2.5 rounded-full ring-2 ring-white ${levelDot[e.level]}`} aria-hidden />
                  <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
                    <time className="font-mono text-xs text-zinc-500" dateTime={e.timestamp} title={new Date(e.timestamp).toLocaleString("en-US")}>
                      {timeOf(e.timestamp)}
                    </time>
                    <span className={e.level === "error" ? "font-medium text-rose-700" : ""}>{describe(e)}</span>
                    {fresh && <span className="text-xs font-medium text-sky-700">new</span>}
                  </div>
                  <div className="mt-1 flex flex-wrap items-center gap-2">
                    <Badge tone={levelTone[e.level]}>{e.source}</Badge>
                    <span className="font-mono text-[11px] text-zinc-400">{e.type}</span>
                  </div>
                </li>
              );
            })}
          </ol>
        </section>
      ))}
    </div>
  );
}

const statusCopy = {
  live: { label: "Live", dot: "bg-emerald-500 animate-pulse", title: "Real-time updates (SSE)" },
  connecting: { label: "Reconnecting…", dot: "bg-amber-500", title: "Reconnecting to stream; updating every 15s in the meantime" },
  polling: { label: "Updating every 15s", dot: "bg-zinc-400", title: "Stream unavailable; using polling" },
} as const;

/** Connection indicator for the realtime stream. */
export function LiveStatus() {
  const s = statusCopy[useRealtimeStatus()];
  return (
    <span className="inline-flex items-center gap-1.5 text-xs text-zinc-500" title={s.title} role="status">
      <span className={`h-2 w-2 rounded-full ${s.dot}`} aria-hidden />
      {s.label}
    </span>
  );
}
