import { describe, expect, it } from "vitest";
import { invalidationsFor, mergeEvent, parseData, reconnectDelay, toTrackerEvent, withEvent } from "./stream";
import type { TrackerEvent } from "./types";

const ev = (id: string, ts: string): TrackerEvent => ({ eventId: id, source: "crosswatc", type: "X", level: "info", timestamp: ts });

describe("parseData", () => {
  it("accepts JSON objects only", () => {
    expect(parseData('{"resourceId":"r1"}')).toEqual({ resourceId: "r1" });
    expect(parseData("[1]")).toBeNull();
    expect(parseData("nope")).toBeNull();
    expect(parseData("null")).toBeNull();
  });
});

describe("invalidationsFor", () => {
  it("refreshes the affected resource views per event type", () => {
    expect(invalidationsFor("resource.updated", { resourceId: "r1" })).toEqual([["resources"], ["resource", "r1"], ["state", "r1"], ["health", "r1"]]);
    expect(invalidationsFor("revision.updated", { resourceId: "r1" })).toContainEqual(["revisions", "r1"]);
    expect(invalidationsFor("topology.updated", { resourceId: "r1" })).toEqual([["topology", "r1"]]);
    expect(invalidationsFor("drift.detected", { resourceId: "r1" })).toContainEqual(["state", "r1"]);
    expect(invalidationsFor("operation.updated", { resourceId: "r1" })).toContainEqual(["revision", "r1"]);
  });
  it("does not refetch the timeline for event.created (patched in place)", () => {
    expect(invalidationsFor("event.created", { resourceId: "r1" })).not.toContainEqual(["events", "r1"]);
  });
  it("falls back to the list when the resource is unknown", () => {
    expect(invalidationsFor("event.created", { resourceId: null })).toEqual([["resources"]]);
  });
});

describe("timeline merge", () => {
  const trigger = { resourceId: "r1", eventId: "e3", source: "github", type: "GITOPS.PR.MERGED", level: "info" as const,
    message: null, createdAt: "2026-10-17T10:05:00.123+00:00" };

  it("maps trigger payloads to timeline entries", () => {
    expect(toTrackerEvent(trigger)).toEqual({ eventId: "e3", source: "github", type: "GITOPS.PR.MERGED", level: "info",
      message: undefined, timestamp: "2026-10-17T10:05:00.123+00:00" });
    expect(toTrackerEvent({ resourceId: "r1" })).toBeNull();
  });

  it("inserts newest-first and de-duplicates", () => {
    const list = [ev("e2", "2026-10-17T10:10:00Z"), ev("e1", "2026-10-17T10:00:00Z")];
    const merged = mergeEvent(list, ev("e3", "2026-10-17T10:05:00Z"));
    expect(merged.map((e) => e.eventId)).toEqual(["e2", "e3", "e1"]);
    expect(mergeEvent(merged, ev("e3", "2026-10-17T10:05:00Z"))).toBe(merged);
    expect(mergeEvent([], ev("a", "2026-10-17T10:00:00Z"))).toHaveLength(1);
    expect(mergeEvent(list, ev("old", "2026-10-16T10:00:00Z")).at(-1)?.eventId).toBe("old");
    expect(mergeEvent(list, ev("new", "2026-10-18T10:00:00Z"))[0].eventId).toBe("new");
  });

  it("patches a cached page without touching unrelated state", () => {
    const page = { items: [ev("e1", "2026-10-17T10:00:00Z")], nextCursor: "c" };
    const out = withEvent(page, trigger);
    expect(out?.items.map((e) => e.eventId)).toEqual(["e3", "e1"]);
    expect(out?.nextCursor).toBe("c");
    expect(withEvent(out, trigger)).toBe(out);
    expect(withEvent(undefined, trigger)).toBeUndefined();
  });
});

describe("reconnectDelay", () => {
  it("backs off exponentially up to a minute", () => {
    expect([0, 1, 2, 3, 4, 10].map(reconnectDelay)).toEqual([5_000, 10_000, 20_000, 40_000, 60_000, 60_000]);
  });
});
