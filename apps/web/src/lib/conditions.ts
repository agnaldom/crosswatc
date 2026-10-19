// Normalizes Crossplane status.conditions into a health tone and a short label.
// Semantics adapted from crossplane-contrib/crossview src/presentation/utils/resourceStatus.js (Apache-2.0).
import type { Tone } from "./labels";

export interface Condition {
  type: string;
  status: "True" | "False" | "Unknown" | string;
  reason?: string;
  message?: string;
}

export interface NormalizedStatus {
  tone: Tone; // ok | warn | error | progress | muted
  label: string;
  detail?: string; // reason/message of the condition that decided the status
  chips: { label: string; tone: Tone }[]; // per-condition badges (Synced, Ready, Healthy…)
}

const find = (cs: Condition[], type: string) => cs.find((c) => c.type === type);
const toneOf = (c?: Condition): Tone => (!c ? "muted" : c.status === "True" ? "ok" : c.status === "False" ? "error" : "warn");
const why = (c?: Condition) => (c && c.status !== "True" ? [c.reason, c.message].filter(Boolean).join(": ") || undefined : undefined);

function chip(c: Condition | undefined, yes: string, no: string) {
  return c ? [{ label: c.status === "True" ? yes : c.status === "False" ? no : `${yes}?`, tone: toneOf(c) }] : [];
}

/** Package kinds (Provider, Function, Configuration) report Installed + Healthy. */
function packageStatus(cs: Condition[]): NormalizedStatus {
  const healthy = find(cs, "Healthy");
  const installed = find(cs, "Installed");
  const chips = [...chip(installed, "Installed", "Not Installed"), ...chip(healthy, "Healthy", "Unhealthy")];
  if (healthy?.status === "True" && installed?.status !== "False") return { tone: "ok", label: "Healthy", chips };
  if (healthy?.status === "False") return { tone: "error", label: "Unhealthy", detail: why(healthy), chips };
  if (installed?.status === "False") return { tone: "error", label: "Not Installed", detail: why(installed), chips };
  if (installed?.status === "True") return { tone: "progress", label: "Installed", chips };
  return { tone: "warn", label: "Pending", chips };
}

/** XRDs report Established (API served) + Offered (claim API served). */
function xrdStatus(cs: Condition[]): NormalizedStatus {
  const est = find(cs, "Established");
  const off = find(cs, "Offered");
  const chips = [...chip(est, "Established", "Not Established"), ...chip(off, "Offered", "Not Offered")];
  if (est?.status === "False") return { tone: "error", label: "Not Established", detail: why(est), chips };
  if (off?.status === "False") return { tone: "error", label: "Not Offered", detail: why(off), chips };
  if (est?.status === "True" && (!off || off.status === "True")) return { tone: "ok", label: off ? "Established & Offered" : "Established", chips };
  if (off?.status === "True" && !est) return { tone: "ok", label: "Offered", chips };
  return { tone: "warn", label: "Pending", chips };
}

/** Claims, XRs and Managed Resources report Synced (reconcile OK) + Ready (resource available). */
function resourceStatus(cs: Condition[]): NormalizedStatus {
  const synced = find(cs, "Synced");
  const ready = find(cs, "Ready");
  const chips = [...chip(synced, "Synced", "Not Synced"), ...chip(ready, "Ready", "Not Ready")];
  if (synced?.status === "False") return { tone: "error", label: "Not Synced", detail: why(synced), chips };
  if (ready?.status === "False") {
    // Ready=False while Synced=True is usually still creating in the cloud
    const creating = /creat|pending|unavailable/i.test(ready.reason ?? "") && synced?.status === "True";
    return { tone: creating ? "progress" : "error", label: creating ? "Creating" : "Not Ready", detail: why(ready), chips };
  }
  if (synced?.status === "True" && ready?.status === "True") return { tone: "ok", label: "Ready", chips };
  if (synced?.status === "True" && !ready) return { tone: "ok", label: "Synced", chips };
  if (ready?.status === "True" && !synced) return { tone: "ok", label: "Ready", chips };
  if (synced || ready) return { tone: "warn", label: "Pending", detail: why(synced ?? ready), chips };
  // other kinds: any True → active, any False → inactive
  if (cs.some((c) => c.status === "False")) return { tone: "error", label: "Inactive", detail: why(cs.find((c) => c.status === "False")), chips };
  if (cs.some((c) => c.status === "True")) return { tone: "ok", label: "Active", chips };
  return { tone: "warn", label: "Pending", chips };
}

const packageKinds = new Set(["Provider", "Function", "Configuration", "ProviderRevision", "FunctionRevision", "ConfigurationRevision"]);

export function normalizeConditions(conditions: Condition[] | undefined | null, kind?: string): NormalizedStatus {
  const cs = conditions ?? [];
  if (cs.length === 0) return { tone: "muted", label: "Unknown", chips: [] };
  if (kind && packageKinds.has(kind)) return packageStatus(cs);
  if (kind === "CompositeResourceDefinition") return xrdStatus(cs);
  return resourceStatus(cs);
}
