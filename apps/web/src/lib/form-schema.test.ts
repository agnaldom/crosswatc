import { describe, expect, it } from "vitest";
import { buildSchema, changedValues } from "./form-schema";
import type { FormField } from "./types";

const base = { path: "", title: "", order: 0, safeInPlace: true, required: false, editable: true };
const fields: FormField[] = [
  { ...base, key: "vmSize", type: "string", enum: ["D2", "D4"], required: true },
  { ...base, key: "diskGb", type: "integer", minimum: 32, maximum: 1024 },
  { ...base, key: "subnet", type: "string", pattern: "^snet-" },
  { ...base, key: "location", type: "string", editable: false },
];

describe("buildSchema", () => {
  const schema = buildSchema(fields);
  it("accepts valid values and ignores read-only fields", () => {
    expect(schema.safeParse({ vmSize: "D4", diskGb: "64", subnet: "snet-a" }).success).toBe(true);
    expect(Object.keys(schema.shape)).not.toContain("location");
  });
  it("enforces enum, bounds, integer and pattern", () => {
    expect(schema.safeParse({ vmSize: "D8" }).success).toBe(false);
    expect(schema.safeParse({ vmSize: "D2", diskGb: 8 }).success).toBe(false);
    expect(schema.safeParse({ vmSize: "D2", diskGb: 40.5 }).success).toBe(false);
    expect(schema.safeParse({ vmSize: "D2", subnet: "vnet" }).success).toBe(false);
  });
});

describe("changedValues", () => {
  it("returns only edited editable keys", () => {
    expect(changedValues(fields, { vmSize: "D2", diskGb: 64 }, { vmSize: "D4", diskGb: "64", location: "x" })).toEqual({ vmSize: "D4" });
  });
});
