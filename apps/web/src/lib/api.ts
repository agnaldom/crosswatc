import type { Health, Page, ResourceStateView, Revision, Topology, TrackedResource, TrackerEvent } from "./types";

export class ApiError extends Error {
  constructor(
    public status: number,
    public title: string,
    public correlationId?: string,
  ) {
    super(title);
  }
}

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`/api/v1${path}`, { headers: { Accept: "application/json" }, cache: "no-store" });
  if (!res.ok) {
    let title = res.statusText;
    let correlationId: string | undefined;
    try {
      const body = await res.json();
      title = body.title ?? title;
      correlationId = body.correlationId;
    } catch {
      /* non-JSON error */
    }
    throw new ApiError(res.status, title, correlationId);
  }
  return res.json() as Promise<T>;
}

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
  topology: (id: string) => get<Topology>(`/resources/${id}/topology`),
};

/** Polling fallback until SSE lands (spec §45). */
export const POLL_INTERVAL_MS = 15_000;
