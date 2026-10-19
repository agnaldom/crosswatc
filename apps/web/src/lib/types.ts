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
  /** List only: observed Claim conditions; absent = unknown. */
  synced?: boolean;
  ready?: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface KindCount {
  kind: string;
  count: number;
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
  drift: DriftState;
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

export type OperationStatus = "QUEUED" | "RUNNING" | "WAITING_APPROVAL" | "SUCCEEDED" | "FAILED" | "CANCELLED";

export interface Operation {
  id: string;
  resourceId: string;
  resourceName?: string;
  revisionId?: string;
  correlationId: string;
  type: "CREATE" | "UPDATE" | "DELETE" | "PLATFORM_UPGRADE" | "RECONCILE";
  status: OperationStatus;
  requestedBy: string;
  requestedAt: string;
  startedAt?: string;
  finishedAt?: string;
}

export interface RevisionDetail extends Revision {
  operations: Operation[];
  events: TrackerEvent[];
}

export interface TrackerEvent {
  eventId: string;
  correlationId?: string;
  revisionId?: string;
  source: string;
  type: string;
  level: "info" | "warning" | "error";
  message?: string;
  timestamp: string;
}

export type NodeStatus = "ready" | "not-ready" | "error" | "missing" | "deleting" | "none";

export interface TopologyNode {
  id: string;
  type: "claim" | "composite" | "managed-resource" | "composition";
  kind: string;
  name: string;
  namespace?: string;
  status: NodeStatus;
  provider?: string;
  externalName?: string;
  message?: string;
  apiVersion?: string;
  conditions?: { type: string; status: string; reason?: string; message?: string; lastTransitionTime?: string }[];
  deletionPolicy?: "Delete" | "Orphan";
  compositionResourceName?: string;
}

export interface TopologyEdge {
  source: string;
  target: string;
}

export interface Topology {
  nodes: TopologyNode[];
  edges: TopologyEdge[];
}

export interface FormField {
  key: string;
  path: string;
  type: "string" | "integer" | "number" | "boolean" | "array" | "object" | "";
  title: string;
  description?: string;
  required: boolean;
  editable: boolean;
  order: number;
  enum?: (string | number)[];
  default?: unknown;
  minimum?: number;
  maximum?: number;
  minLength?: number;
  maxLength?: number;
  pattern?: string;
  immutable?: boolean;
  safeInPlace: boolean;
  approvalRequired?: Record<string, boolean>;
  group?: string;
}

export interface ResourceForm {
  form: { kind: string; version: string; fields: FormField[] };
  values: Record<string, unknown>;
  editable: boolean;
  reason?: string;
}

export interface SpecChange {
  field: string;
  path: string;
  from: unknown;
  to: unknown;
  impact: "IN_PLACE" | "REPLACEMENT";
}

export interface ChangePreview {
  valid: boolean;
  changes: SpecChange[];
  policy: { result: "PASS" | "FAIL"; violations: { field?: string; rule: string; message: string }[] };
  impact: { replacementRequired: boolean; possibleDowntime: boolean; approvalRequired: boolean };
  warnings: string[];
  cost?: CostDiff;
}

export interface CostEstimate {
  monthly: number;
  currency: string;
  lines: { component: string; sku: string; region: string; unitPrice: number; quantity: number; monthly: number }[];
  unknown?: string[];
}

export interface CostDiff {
  current: CostEstimate;
  new: CostEstimate;
  monthlyDelta: number;
  currency: string;
  partial: boolean;
}

export interface ChangeCreated {
  operationId: string;
  revisionId: string;
  rev: number;
  correlationId: string;
  prNumber: number;
  prUrl: string;
  branch: string;
}

export interface AuditEntry {
  id: number;
  actor: string;
  action: "change.requested" | "change.failed" | "revision.cancelled" | "revision.superseded" | string;
  resourceId?: string;
  revisionId?: string;
  operationId?: string;
  correlationId?: string;
  before?: Record<string, unknown>;
  after?: Record<string, unknown>;
  prUrl?: string;
  commitSha?: string;
  createdAt: string;
}

export interface DriftState {
  detected: boolean;
  signals?: string[];
  differences?: { field: string; desired: unknown; observed: unknown }[];
  detectedAt?: string;
}

export interface DecommissionPreview {
  allowed: boolean;
  reason?: string;
  environment: string;
  approvalRequired: boolean;
  confirmName: string;
  deleted: { kind: string; name: string; provider?: string; externalName?: string }[];
  orphaned: { kind: string; name: string; provider?: string; externalName?: string }[];
  warnings: string[];
}

export interface KubeEvent {
  object: { apiVersion?: string; kind: string; name: string; namespace?: string };
  type: "Normal" | "Warning" | string;
  reason: string;
  message: string;
  count: number;
  source?: string;
  firstSeen?: string;
  lastSeen?: string;
}

export interface ManifestObject {
  id: string;
  type: "claim" | "composite" | "managed-resource" | "composition";
  kind: string;
  name: string;
  namespace?: string;
}

export interface Me {
  user: string;
  squads: string[];
  sre: boolean;
}

export interface CrossplanePackage {
  kind: "Provider" | "Function" | "Configuration";
  name: string;
  package: string;
  currentRevision?: string;
  currentIdentifier?: string;
  revisionActivationPolicy?: string;
  conditions: { type: string; status: string; reason?: string; message?: string }[];
}

export interface CompositionSummary {
  name: string;
  compositeType: string;
  mode: "Pipeline" | "Resources" | string;
  functions?: string[];
  steps: number;
  latestRevision?: string;
}

export interface PlatformXRD {
  name: string;
  group: string;
  kind: string;
  claimKind?: string;
  scope: string;
  version: string;
  established: boolean;
}
