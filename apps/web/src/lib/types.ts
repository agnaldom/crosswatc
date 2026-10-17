// Mirrors the Go API DTOs (internal/httpapi/dto.go).

export type ResourceState =
  | "PROVISIONING"
  | "OPERATIONAL"
  | "UPDATING"
  | "DEGRADED"
  | "DELETE_PENDING"
  | "DELETING"
  | "DELETED"
  | "FAILED"
  | "UNKNOWN";

export type RevisionState =
  | "CREATED"
  | "VALIDATING"
  | "WAITING_APPROVAL"
  | "MERGED"
  | "SYNCING"
  | "RECONCILING"
  | "APPLYING"
  | "SUCCEEDED"
  | "FAILED"
  | "REJECTED"
  | "CANCELLED"
  | "SUPERSEDED";

export interface Page<T> {
  items: T[];
  nextCursor: string | null;
}

export interface TrackedResource {
  id: string;
  name: string;
  kind: string;
  provider: string;
  environment: string;
  owner: string;
  requestedBy?: string;
  region?: string;
  resourceGroup?: string;
  claimRef: { namespace: string; name: string; kind: string; apiVersion: string };
  repository: string;
  gitPath: string;
  resourceState: ResourceState;
  currentRevision?: number;
  composition?: { name: string; revision?: string };
  correlationId: string;
  createdAt: string;
  updatedAt: string;
}

export type Stage = "REQUEST" | "PULL_REQUEST" | "CI_POLICIES" | "ARGO_CD" | "CROSSPLANE" | "CLOUD" | "OPERATIONAL";
export type StageStatus = "PENDING" | "IN_PROGRESS" | "SUCCEEDED" | "FAILED" | "SKIPPED";

export interface StageView {
  stage: Stage;
  status: StageStatus;
  actionOwner?: "requester" | "approver" | "platform" | "squad";
  message?: string;
  technicalDetail?: string;
}

export interface ResourceStateView {
  resourceState: ResourceState;
  currentRevision: { rev: number; state: RevisionState; prUrl?: string } | null;
  stages: StageView[];
  drift: { detected: boolean };
  lastReconcileAt: string | null;
}

export type HealthValue = "ok" | "attention" | "unknown";
export type Health = Partial<Record<"crossplane" | "argocd" | "cloud" | "configuration" | "secrets", HealthValue>>;

export interface Revision {
  id: string;
  resourceId: string;
  rev: number;
  kind: "CREATE" | "UPDATE" | "DELETE" | "PLATFORM_UPGRADE";
  state: RevisionState;
  commitSha?: string;
  prNumber?: number;
  prUrl?: string;
  failureReason?: { technicalCode: string; userMessage: string; recommendedAction?: string; runbook?: string };
  startedAt?: string;
  completedAt?: string;
}

export interface TrackerEvent {
  eventId: string;
  correlationId?: string;
  source: string;
  type: string;
  level: "info" | "warning" | "error";
  message?: string;
  timestamp: string;
}

export type NodeStatus = "ready" | "not-ready" | "error" | "missing" | "deleting";

export interface TopologyNode {
  id: string;
  type: "claim" | "composite" | "managed-resource";
  kind: string;
  name: string;
  namespace?: string;
  status: NodeStatus;
  provider?: string;
  externalName?: string;
  message?: string;
}

export interface TopologyEdge {
  source: string;
  target: string;
}

export interface Topology {
  nodes: TopologyNode[];
  edges: TopologyEdge[];
}
