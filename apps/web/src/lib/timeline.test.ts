import { describe as suite, expect, it } from "vitest";
import { describe, eventLabel, groupByDay, isFresh, timeOf } from "./timeline";
import type { TrackerEvent } from "./types";

const ev = (id: string, ts: string, extra: Partial<TrackerEvent> = {}): TrackerEvent => ({
  eventId: id, source: "crosswatc", type: "REQUEST.CREATED", level: "info", timestamp: ts, ...extra,
});

suite("timeline", () => {
  it("groups newest-first events by local day", () => {
    const local = (d: number, h: number) => new Date(2026, 9, d, h, 30).toISOString();
    const groups = groupByDay([ev("c", local(18, 9)), ev("b", local(17, 22)), ev("a", local(17, 7))]);
    expect(groups.map((g) => [g.day, g.items.map((e) => e.eventId)])).toEqual([
      ["2026-10-18", ["c"]],
      ["2026-10-17", ["b", "a"]],
    ]);
    expect(groupByDay([])).toEqual([]);
  });

  it("formats HH:mm in local time", () => {
    expect(timeOf(new Date(2026, 9, 17, 7, 5).toISOString())).toBe("07:05");
  });

  it("prefers the message, then the canonical label, then the raw type", () => {
    expect(describe(ev("a", "2026-10-17T10:00:00Z", { message: "Postgres resource discovered" }))).toBe("Postgres resource discovered");
    expect(describe(ev("a", "2026-10-17T10:00:00Z"))).toBe(eventLabel["REQUEST.CREATED"]);
    expect(describe(ev("a", "2026-10-17T10:00:00Z", { type: "CUSTOM.THING" }))).toBe("CUSTOM.THING");
  });

  it("labels every canonical type of §33", () => {
    expect(Object.keys(eventLabel)).toHaveLength(27);
  });

  it("highlights only recent events", () => {
    const now = Date.parse("2026-10-17T10:00:20Z");
    expect(isFresh("2026-10-17T10:00:10Z", now)).toBe(true);
    expect(isFresh("2026-10-17T09:59:00Z", now)).toBe(false);
    expect(isFresh("2026-10-17T10:01:00Z", now)).toBe(false); // clock skew: future
    expect(isFresh("garbage", now)).toBe(false);
  });
});
