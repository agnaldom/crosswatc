import { describe, expect, it } from "vitest";
import { splitMatches, tokenizeLine } from "./yaml-highlight";

const kinds = (l: string) => tokenizeLine(l).map((t) => `${t.kind}:${t.text}`);

describe("tokenizeLine", () => {
  it("key/value scalars", () => {
    expect(kinds("apiVersion: v1")).toEqual(["key:apiVersion", "colon:: ", "string:v1"]);
    expect(kinds("  replicas: 3")).toEqual(["indent:  ", "key:replicas", "colon:: ", "number:3"]);
    expect(kinds("ready: true # ok")).toEqual(["key:ready", "colon:: ", "bool:true", "comment: # ok"]);
    expect(kinds("owner: null")).toEqual(["key:owner", "colon:: ", "null:null"]);
    expect(kinds("metadata:")).toEqual(["key:metadata", "colon::"]);
  });
  it("lists, urls and comments", () => {
    expect(kinds("- name: srv")).toEqual(["dash:- ", "key:name", "colon:: ", "string:srv"]);
    expect(kinds("- Delete")).toEqual(["dash:- ", "string:Delete"]);
    expect(kinds("url: https://x.io/a:b")).toEqual(["key:url", "colon:: ", "string:https://x.io/a:b"]);
    expect(kinds("# header")).toEqual(["comment:# header"]);
    expect(kinds("\"crosswatc.io/owner\": payments")).toEqual(["key:\"crosswatc.io/owner\"", "colon:: ", "string:payments"]);
  });
});

describe("splitMatches", () => {
  it("case-insensitive, all occurrences", () => {
    expect(splitMatches("Ready ready", "ready")).toEqual([
      { text: "Ready", hit: true }, { text: " ", hit: false }, { text: "ready", hit: true },
    ]);
    expect(splitMatches("abc", "")).toEqual([{ text: "abc", hit: false }]);
  });
});
