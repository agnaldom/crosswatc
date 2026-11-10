import { expect, it } from "vitest";
import { GET } from "../app/platform/route";

it("redirects old bookmarks to independent pages before rendering or checking authentication", () => {
  const cases = [
    ["Providers", "/providers"],
    ["Functions", "/functions"],
    ["Configurations", "/configurations"],
    ["Compositions", "/compositions"],
    ["XRDs", "/resource-definitions"],
    ["MRDs", "/managed-resource-definitions"],
    ["Activation Policies", "/activation-policies"],
  ];
  for (const [tab, destination] of cases) {
    const result = GET(new Request(`https://workspace.example/platform?${new URLSearchParams({ tab })}`));
    expect(result.status).toBe(307);
    expect(result.headers.get("Location")).toBe(destination);
  }
  for (const query of ["", "?tab=unknown", "?tab=__proto__", "?tab=https://example.com", "?tab=Functions&tab=Providers"]) {
    const result = GET(new Request(`https://workspace.example/platform${query}`));
    expect(result.status).toBe(307);
    expect(result.headers.get("Location")).toBe("/providers");
  }
});
