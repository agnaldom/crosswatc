/** One-click list filters (Crossview QuickFilters), expressed as URL params so they can be linked. */
export const quickFilters = [
  { id: "failed", label: "Com falha", param: "status", value: "FAILED", tone: "error" },
  { id: "not-ready", label: "Not Ready", param: "ready", value: "false", tone: "error" },
  { id: "not-synced", label: "Not Synced", param: "synced", value: "false", tone: "warn" },
] as const;

export type QuickFilter = (typeof quickFilters)[number];

export function isQuickActive(params: URLSearchParams, f: QuickFilter): boolean {
  return params.get(f.param) === f.value;
}

/** Toggles a quick filter, keeping the other params (kind, q, …) and replacing any other value of the same param. */
export function toggleQuick(params: URLSearchParams, f: QuickFilter): string {
  const next = new URLSearchParams(params);
  if (isQuickActive(params, f)) next.delete(f.param);
  else next.set(f.param, f.value);
  return next.toString();
}
