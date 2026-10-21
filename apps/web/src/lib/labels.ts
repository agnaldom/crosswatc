import type { HealthValue, ResourceState, RevisionState, Stage, StageStatus } from "./types";

export const resourceStateLabel: Record<ResourceState, string> = {
  PROVISIONING: "Provisioning",
  OPERATIONAL: "Operational",
  UPDATING: "Updating",
  DEGRADED: "Degraded",
  DELETE_PENDING: "Delete pending",
  DELETING: "Deleting",
  DELETED: "Deleted",
  FAILED: "Failed",
  UNKNOWN: "Unknown",
};

export type Tone = "ok" | "progress" | "warn" | "error" | "muted";

export const resourceStateTone: Record<ResourceState, Tone> = {
  PROVISIONING: "progress",
  OPERATIONAL: "ok",
  UPDATING: "progress",
  DEGRADED: "warn",
  DELETE_PENDING: "warn",
  DELETING: "progress",
  DELETED: "muted",
  FAILED: "error",
  UNKNOWN: "muted",
};

export const revisionStateTone = (s: RevisionState): Tone =>
  s === "SUCCEEDED" ? "ok" : s === "FAILED" || s === "REJECTED" ? "error" : s === "CANCELLED" || s === "SUPERSEDED" ? "muted" : "progress";

export const stageLabel: Record<Stage, string> = {
  REQUEST: "Request",
  PULL_REQUEST: "Pull Request",
  CI_POLICIES: "CI & Policies",
  ARGO_CD: "Argo CD",
  CROSSPLANE: "Crossplane",
  CLOUD: "Cloud",
  OPERATIONAL: "Operational",
};

export const stageTone: Record<StageStatus, Tone> = {
  PENDING: "muted",
  IN_PROGRESS: "progress",
  SUCCEEDED: "ok",
  FAILED: "error",
  SKIPPED: "muted",
};

export const actionOwnerLabel = {
  requester: "Requester",
  approver: "Approver",
  platform: "Platform",
  squad: "Squad",
} as const;

export const healthTone: Record<HealthValue, Tone> = { ok: "ok", attention: "warn", unknown: "muted" };

export const toneClasses: Record<Tone, string> = {
  ok: "bg-emerald-50 text-emerald-700 ring-emerald-600/20",
  progress: "bg-sky-50 text-sky-700 ring-sky-600/20",
  warn: "bg-amber-50 text-amber-800 ring-amber-600/20",
  error: "bg-rose-50 text-rose-700 ring-rose-600/20",
  muted: "bg-zinc-100 text-zinc-600 ring-zinc-500/20",
};

export const toneDot: Record<Tone, string> = {
  ok: "bg-emerald-500",
  progress: "bg-sky-500",
  warn: "bg-amber-500",
  error: "bg-rose-500",
  muted: "bg-zinc-400",
};
