import type {
  CompositionSummary,
  CrossplanePackage,
  Me,
  PlatformXRD,
  PlatformMRD,
  PlatformMRAP,
  V2List,
  ManifestObject,
  KubeEvent,
  DecommissionPreview,
  AuditEntry,
  ChangeCreated,
  ChangePreview,
  Health,
  ResourceForm,
  Operation,
  Page,
  ResourceStateView,
  Revision,
  RevisionDetail,
  Topology,
  KindCount,
  DashboardSummary,
  TrackedResource,
  TrackerEvent,
} from "./types";

export class ApiError extends Error {
  constructor(
    public status: number,
    public title: string,
    public correlationId?: string,
  ) {
    super(title);
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`/api/v1${path}`, { ...init, headers: { Accept: "application/json", ...init?.headers }, cache: "no-store" });
  if (!res.ok) {
    let title = res.statusText;
    let correlationId: string | undefined;
    let detail: string | undefined;
    try {
      const body = await res.json();
      title = body.title ?? title;
      detail = body.detail;
      correlationId = body.correlationId;
    } catch {
      /* non-JSON error */
    }
    throw new ApiError(res.status, detail ? `${title}: ${detail}` : title, correlationId);
  }
  return res.json() as Promise<T>;
}

const get = <T>(path: string) => request<T>(path);
const post = <T>(path: string, body: unknown) =>
  request<T>(path, { method: "POST", body: JSON.stringify(body), headers: { "Content-Type": "application/json" } });

export interface ResourceFilters {
  environment?: string;
  provider?: string;
  status?: string;
  kind?: string;
  /** "true" | "false" | "unknown" */
  synced?: string;
  ready?: string;
  q?: string;
  limit?: number;
  cursor?: string;
}

export const api = {
  listResources: (f: ResourceFilters = {}) => {
    const qs = new URLSearchParams(Object.entries(f).filter(([, v]) => v).map(([k, v]) => [k, String(v)]));
    return get<Page<TrackedResource>>(`/resources${qs.size ? `?${qs}` : ""}`);
  },
  dashboard: () => get<DashboardSummary>("/dashboard"),
  resourceKinds: () => get<{ items: KindCount[] }>("/resources/kinds"),
  resource: (id: string) => get<TrackedResource>(`/resources/${id}`),
  state: (id: string) => get<ResourceStateView>(`/resources/${id}/state`),
  health: (id: string) => get<Health>(`/resources/${id}/health`),
  events: (id: string) => get<Page<TrackerEvent>>(`/resources/${id}/events?limit=100`),
  revisions: (id: string) => get<Page<Revision>>(`/resources/${id}/revisions`),
  revision: (id: string, rev: number) => get<RevisionDetail>(`/resources/${id}/revisions/${rev}`),
  operations: (id: string) => get<Page<Operation>>(`/resources/${id}/operations`),
  topology: (id: string) => get<Topology>(`/resources/${id}/topology`),
  form: (id: string) => get<ResourceForm>(`/resources/${id}/form`),
  me: () => get<Me>("/me"),
  platformPackages: (kind: "providers" | "functions" | "configurations") => get<Page<CrossplanePackage>>(`/platform/${kind}`),
  platformXRDs: () => get<Page<PlatformXRD>>("/platform/xrds"),
  platformMRDs: () => get<V2List<PlatformMRD>>("/platform/mrds"),
  platformMRAPs: () => get<V2List<PlatformMRAP>>("/platform/mraps"),
  platformCompositions: () => get<Page<CompositionSummary>>("/platform/compositions"),
  manifest: (id: string, object?: string) =>
    get<{ object: ManifestObject; yaml: string; objects: ManifestObject[] }>(
      `/resources/${id}/manifest${object ? `?object=${encodeURIComponent(object)}` : ""}`,
    ),
  kubeEvents: (id: string) =>
    get<{ items: KubeEvent[]; errors: { object: { kind: string; name: string }; error: string }[] }>(`/resources/${id}/kube-events`),
  audit: (id: string) => get<Page<AuditEntry>>(`/resources/${id}/audit?limit=100`),
  preview: (id: string, changes: Record<string, unknown>) => post<ChangePreview>(`/resources/${id}/changes/preview`, { changes }),
  decommissionPreview: (id: string) => post<DecommissionPreview>(`/resources/${id}/decommission/preview`, {}),
  decommission: (id: string, confirmName: string, justification: string) =>
    post<ChangeCreated>(`/resources/${id}/decommission`, { confirmName, justification }),
  cancelRevision: (id: string, rev: number, reason: string) =>
    fetch(`/api/v1/resources/${id}/revisions/${rev}/cancel`, {
      method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ reason }),
    }).then(async (r) => {
      if (!r.ok) {
        const b = await r.json().catch(() => ({}));
        throw new ApiError(r.status, b.detail ? `${b.title}: ${b.detail}` : (b.title ?? r.statusText), b.correlationId);
      }
    }),
  createChange: (id: string, changes: Record<string, unknown>, justification: string, supersede = false) =>
    post<ChangeCreated>(`/resources/${id}/changes`, { changes, justification, supersede }),
};

/** Polling interval used only while the realtime stream is unavailable (spec §45). */
export const POLL_INTERVAL_MS = 15_000;
