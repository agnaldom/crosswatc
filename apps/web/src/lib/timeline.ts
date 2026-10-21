// Event timeline helpers (spec §46 Events).
import type { TrackerEvent } from "./types";

/** Human labels for canonical event types (spec §33), used when an event has no message. */
export const eventLabel: Record<string, string> = {
  "REQUEST.CREATED": "Request received",
  "REQUEST.SCAFFOLDER.COMPLETED": "Configuration validated",
  "REQUEST.SCAFFOLDER.FAILED": "Backstage scaffolding failed",
  "GITOPS.PR.CREATED": "Pull Request opened",
  "GITOPS.PR.CLOSED": "Pull Request closed without merge",
  "GITOPS.CHECK.SUCCEEDED": "Validations passed",
  "GITOPS.CHECK.FAILED": "Validations failed",
  "GITOPS.PR.APPROVED": "Pull Request approved",
  "GITOPS.PR.MERGED": "Pull Request merged",
  "GITOPS.SYNC.STARTED": "Argo CD sync started",
  "GITOPS.SYNC.SUCCEEDED": "Argo CD sync succeeded",
  "GITOPS.SYNC.FAILED": "Argo CD sync failed",
  "INFRA.CLAIM.CREATED": "Claim discovered",
  "INFRA.CLAIM.DELETED": "Claim deleted",
  "INFRA.CLAIM.INVALID_METADATA": "Claim metadata invalid",
  "INFRA.XR.CREATED": "Composite created",
  "INFRA.RECONCILE.STARTED": "Crossplane reconcile started",
  "INFRA.RECONCILE.READY": "Crossplane reconciled",
  "INFRA.RECONCILE.FAILED": "Crossplane reconcile failed",
  "INFRA.MANAGED_RESOURCE.CREATED": "Cloud resource created",
  "INFRA.MANAGED_RESOURCE.READY": "Cloud resource ready",
  "INFRA.MANAGED_RESOURCE.FAILED": "Cloud resource failed",
  "LIFECYCLE.OPERATIONAL": "Resource operational",
  "LIFECYCLE.DEGRADED": "Resource degraded",
  "LIFECYCLE.DRIFT.DETECTED": "Drift detected",
  "LIFECYCLE.DRIFT.RESOLVED": "Drift resolved",
  "OPERATION.STUCK": "Operation stuck without progress",
};

export function describe(e: TrackerEvent): string {
  return e.message || eventLabel[e.type] || e.type;
}

export interface DayGroup {
  day: string; // YYYY-MM-DD in the viewer's timezone
  items: TrackerEvent[];
}

function localDay(ts: string): string {
  const d = new Date(ts);
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

/** Groups newest-first events by local day, preserving order. */
export function groupByDay(events: TrackerEvent[]): DayGroup[] {
  const out: DayGroup[] = [];
  for (const e of events) {
    const day = localDay(e.timestamp);
    const last = out.at(-1);
    if (last?.day === day) last.items.push(e);
    else out.push({ day, items: [e] });
  }
  return out;
}

/** HH:mm in the viewer's timezone (spec §46: "07:31 Request received"). */
export function timeOf(ts: string): string {
  const d = new Date(ts);
  return `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
}

/** Events newer than windowMs are highlighted as just arrived. */
export function isFresh(ts: string, now: number, windowMs = 15_000): boolean {
  const t = Date.parse(ts);
  return !Number.isNaN(t) && now - t >= 0 && now - t < windowMs;
}
