// Realtime stream (spec §45): maps SSE messages from GET /api/v1/stream to TanStack Query cache updates.
// Pure functions only — the EventSource wiring lives in realtime.tsx.
import type { QueryKey } from "@tanstack/react-query";
import type { Page, TrackerEvent } from "./types";

export const STREAM_TYPES = [
  "resource.updated",
  "revision.updated",
  "operation.updated",
  "event.created",
  "topology.updated",
  "drift.detected",
] as const;

export type StreamEventType = (typeof STREAM_TYPES)[number];

/** Payload written by the database triggers (db/migrations/00003_stream_event.sql). */
export interface StreamData {
  resourceId?: string | null;
  eventId?: string;
  revisionId?: string | null;
  revision?: number;
  state?: string;
  status?: string;
  source?: string;
  type?: string;
  level?: TrackerEvent["level"];
  message?: string | null;
  createdAt?: string;
}

export function parseData(raw: string): StreamData | null {
  try {
    const v = JSON.parse(raw);
    return v && typeof v === "object" && !Array.isArray(v) ? (v as StreamData) : null;
  } catch {
    return null;
  }
}

/** Query keys (prefixes) to refetch after a stream message. Lists are always refreshed. */
export function invalidationsFor(type: StreamEventType, d: StreamData): QueryKey[] {
  const id = d.resourceId;
  if (!id) return [["resources"]];
  switch (type) {
    case "resource.updated":
      return [["resources"], ["resource", id], ["state", id], ["health", id]];
    case "revision.updated":
      return [["resources"], ["state", id], ["revisions", id], ["revision", id]];
    case "operation.updated":
      return [["operations"], ["revision", id]];
    case "event.created":
      return [["revision", id]]; // the resource timeline itself is updated in place (withEvent)
    case "topology.updated":
      return [["topology", id]];
    case "drift.detected":
      return [["resource", id], ["state", id]];
  }
}

/** Converts an event.created payload into a timeline entry. */
export function toTrackerEvent(d: StreamData): TrackerEvent | null {
  if (!d.eventId || !d.type || !d.source || !d.createdAt) return null;
  return {
    eventId: d.eventId,
    source: d.source,
    type: d.type,
    level: d.level ?? "info",
    message: d.message ?? undefined,
    timestamp: d.createdAt,
  };
}

/** Inserts e into a newest-first list, de-duplicated by eventId. */
export function mergeEvent(list: TrackerEvent[], e: TrackerEvent): TrackerEvent[] {
  if (list.some((x) => x.eventId === e.eventId)) return list;
  const t = Date.parse(e.timestamp);
  const i = list.findIndex((x) => Date.parse(x.timestamp) <= t);
  return i === -1 ? [...list, e] : [...list.slice(0, i), e, ...list.slice(i)];
}

/** Applies an event.created message to a cached events page. */
export function withEvent(page: Page<TrackerEvent> | undefined, d: StreamData): Page<TrackerEvent> | undefined {
  const e = toTrackerEvent(d);
  if (!page || !e) return page;
  const items = mergeEvent(page.items, e);
  return items === page.items ? page : { ...page, items };
}

/** Reconnect delay after the stream is closed by the server (5s, 10s, … capped at 60s). */
export function reconnectDelay(attempt: number): number {
  return Math.min(5_000 * 2 ** attempt, 60_000);
}
