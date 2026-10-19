// Event timeline helpers (spec §46 Events).
import type { TrackerEvent } from "./types";

/** Human labels for canonical event types (spec §33), used when an event has no message. */
export const eventLabel: Record<string, string> = {
  "REQUEST.CREATED": "Solicitação recebida",
  "REQUEST.SCAFFOLDER.COMPLETED": "Configuração validada",
  "REQUEST.SCAFFOLDER.FAILED": "Falha no Backstage",
  "GITOPS.PR.CREATED": "Pull Request criado",
  "GITOPS.PR.CLOSED": "Pull Request fechado sem merge",
  "GITOPS.CHECK.SUCCEEDED": "Validações aprovadas",
  "GITOPS.CHECK.FAILED": "Validações falharam",
  "GITOPS.PR.APPROVED": "Pull Request aprovado",
  "GITOPS.PR.MERGED": "Pull Request mergeado",
  "GITOPS.SYNC.STARTED": "Argo CD sync iniciado",
  "GITOPS.SYNC.SUCCEEDED": "Argo CD sync concluído",
  "GITOPS.SYNC.FAILED": "Argo CD sync falhou",
  "INFRA.CLAIM.CREATED": "Claim descoberto",
  "INFRA.CLAIM.DELETED": "Claim removido",
  "INFRA.CLAIM.INVALID_METADATA": "Metadata do Claim inválida",
  "INFRA.XR.CREATED": "Composite criado",
  "INFRA.RECONCILE.STARTED": "Reconciliação do Crossplane iniciada",
  "INFRA.RECONCILE.READY": "Crossplane reconciliado",
  "INFRA.RECONCILE.FAILED": "Reconciliação do Crossplane falhou",
  "INFRA.MANAGED_RESOURCE.CREATED": "Recurso de cloud criado",
  "INFRA.MANAGED_RESOURCE.READY": "Recurso de cloud atualizado",
  "INFRA.MANAGED_RESOURCE.FAILED": "Recurso de cloud com falha",
  "LIFECYCLE.OPERATIONAL": "Recurso operacional",
  "LIFECYCLE.DEGRADED": "Recurso degradado",
  "LIFECYCLE.DRIFT.DETECTED": "Drift detectado",
  "LIFECYCLE.DRIFT.RESOLVED": "Drift resolvido",
  "OPERATION.STUCK": "Operação sem progresso",
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
