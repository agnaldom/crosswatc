import type {
  ChangePreview,
  Health,
  ResourceForm,
  Operation,
  Page,
  ResourceStateView,
  Revision,
  RevisionDetail,
  Topology,
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
  q?: string;
  cursor?: string;
}

export const api = {
  listResources: (f: ResourceFilters = {}) => {
    const qs = new URLSearchParams(Object.entries(f).filter(([, v]) => v) as [string, string][]);
    return get<Page<TrackedResource>>(`/resources${qs.size ? `?${qs}` : ""}`);
  },
  resource: (id: string) => get<TrackedResource>(`/resources/${id}`),
  state: (id: string) => get<ResourceStateView>(`/resources/${id}/state`),
  health: (id: string) => get<Health>(`/resources/${id}/health`),
  events: (id: string) => get<Page<TrackerEvent>>(`/resources/${id}/events?limit=100`),
  revisions: (id: string) => get<Page<Revision>>(`/resources/${id}/revisions`),
  revision: (id: string, rev: number) => get<RevisionDetail>(`/resources/${id}/revisions/${rev}`),
  operations: (id: string) => get<Page<Operation>>(`/resources/${id}/operations`),
  topology: (id: string) => get<Topology>(`/resources/${id}/topology`),
  form: (id: string) => get<ResourceForm>(`/resources/${id}/form`),
  preview: (id: string, changes: Record<string, unknown>) => post<ChangePreview>(`/resources/${id}/changes/preview`, { changes }),
};

/** Polling interval used only while the realtime stream is unavailable (spec §45). */
export const POLL_INTERVAL_MS = 15_000;
