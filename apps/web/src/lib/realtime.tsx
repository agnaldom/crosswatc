"use client";

// EventSource bridge for GET /api/v1/stream (spec §45). While the stream is live, query polling is off and
// SSE messages invalidate/patch the TanStack Query cache; when it is down, queries fall back to 15s polling.
import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useSyncExternalStore } from "react";
import type { Page, TrackerEvent } from "./types";
import { STREAM_TYPES, invalidationsFor, parseData, reconnectDelay, withEvent, type StreamEventType } from "./stream";

export type RealtimeStatus = "connecting" | "live" | "polling";

let status: RealtimeStatus = "polling";
const listeners = new Set<() => void>();

function setStatus(s: RealtimeStatus) {
  if (s === status) return;
  status = s;
  listeners.forEach((l) => l());
}

export function getRealtimeStatus(): RealtimeStatus {
  return status;
}

function subscribe(l: () => void) {
  listeners.add(l);
  return () => listeners.delete(l);
}

export function useRealtimeStatus(): RealtimeStatus {
  return useSyncExternalStore(subscribe, getRealtimeStatus, () => "polling");
}

/** Mounts the single app-wide stream connection. */
export function RealtimeBridge() {
  const qc = useQueryClient();

  useEffect(() => {
    if (typeof EventSource === "undefined") return; // keep polling
    let es: EventSource | null = null;
    let lastId = "";
    let attempt = 0;
    let opened = false;
    let timer: ReturnType<typeof setTimeout> | undefined;

    const handle = (type: StreamEventType, ev: MessageEvent<string>) => {
      if (ev.lastEventId) lastId = ev.lastEventId;
      const d = parseData(ev.data);
      if (!d) return;
      if (type === "event.created" && d.resourceId) {
        qc.setQueriesData<Page<TrackerEvent>>({ queryKey: ["events", d.resourceId] }, (old) => withEvent(old, d));
      }
      for (const queryKey of invalidationsFor(type, d)) void qc.invalidateQueries({ queryKey });
    };

    const open = () => {
      setStatus("connecting");
      // The browser resends Last-Event-ID on its own reconnects; after a manual reopen we pass it explicitly.
      es = new EventSource(`/api/v1/stream${lastId ? `?lastEventId=${encodeURIComponent(lastId)}` : ""}`);
      es.onopen = () => {
        attempt = 0;
        setStatus("live");
        if (opened) void qc.invalidateQueries(); // catch up on anything older than the replay window
        opened = true;
      };
      es.onerror = () => {
        if (es?.readyState === EventSource.CLOSED) {
          es.close();
          setStatus("polling");
          timer = setTimeout(open, reconnectDelay(attempt++));
        } else {
          setStatus("connecting"); // browser is retrying; polling covers the gap
        }
      };
      for (const t of STREAM_TYPES) es.addEventListener(t, (ev) => handle(t, ev as MessageEvent<string>));
    };

    open();
    return () => {
      clearTimeout(timer);
      es?.close();
      setStatus("polling");
    };
  }, [qc]);

  return null;
}
