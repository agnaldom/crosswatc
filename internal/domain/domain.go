// Package domain holds Crosswatc domain enums and value types (spec §11–§19, §31).
package domain

// ResourceState is the derived state of a tracked resource (§12).
type ResourceState string

const (
	ResourceProvisioning  ResourceState = "PROVISIONING"
	ResourceOperational   ResourceState = "OPERATIONAL"
	ResourceUpdating      ResourceState = "UPDATING"
	ResourceDegraded      ResourceState = "DEGRADED"
	ResourceDeletePending ResourceState = "DELETE_PENDING"
	ResourceDeleting      ResourceState = "DELETING"
	ResourceDeleted       ResourceState = "DELETED"
	ResourceFailed        ResourceState = "FAILED"
	ResourceUnknown       ResourceState = "UNKNOWN"
)

// RevisionKind is the type of change a revision carries (§13).
type RevisionKind string

const (
	RevisionCreate          RevisionKind = "CREATE"
	RevisionUpdate          RevisionKind = "UPDATE"
	RevisionDelete          RevisionKind = "DELETE"
	RevisionPlatformUpgrade RevisionKind = "PLATFORM_UPGRADE"
)

// RevisionState is the lifecycle state of a change (§14).
type RevisionState string

const (
	RevisionCreated         RevisionState = "CREATED"
	RevisionValidating      RevisionState = "VALIDATING"
	RevisionWaitingApproval RevisionState = "WAITING_APPROVAL"
	RevisionMerged          RevisionState = "MERGED"
	RevisionSyncing         RevisionState = "SYNCING"
	RevisionReconciling     RevisionState = "RECONCILING"
	RevisionApplying        RevisionState = "APPLYING"
	RevisionSucceeded       RevisionState = "SUCCEEDED"
	RevisionFailed          RevisionState = "FAILED"
	RevisionRejected        RevisionState = "REJECTED"
	RevisionCancelled       RevisionState = "CANCELLED"
	RevisionSuperseded      RevisionState = "SUPERSEDED"
)

// Terminal reports whether no further transition is allowed.
func (s RevisionState) Terminal() bool {
	switch s {
	case RevisionSucceeded, RevisionFailed, RevisionRejected, RevisionCancelled, RevisionSuperseded:
		return true
	}
	return false
}

// OperationType (§15).
type OperationType string

const (
	OperationCreate          OperationType = "CREATE"
	OperationUpdate          OperationType = "UPDATE"
	OperationDelete          OperationType = "DELETE"
	OperationPlatformUpgrade OperationType = "PLATFORM_UPGRADE"
	OperationReconcile       OperationType = "RECONCILE"
)

// OperationStatus (§15).
type OperationStatus string

const (
	OperationQueued          OperationStatus = "QUEUED"
	OperationRunning         OperationStatus = "RUNNING"
	OperationWaitingApproval OperationStatus = "WAITING_APPROVAL"
	OperationSucceeded       OperationStatus = "SUCCEEDED"
	OperationFailed          OperationStatus = "FAILED"
	OperationCancelled       OperationStatus = "CANCELLED"
)

// OperationStatusFor maps a revision state to the operation status (§15).
func OperationStatusFor(s RevisionState) OperationStatus {
	switch s {
	case RevisionCreated:
		return OperationQueued
	case RevisionWaitingApproval:
		return OperationWaitingApproval
	case RevisionSucceeded:
		return OperationSucceeded
	case RevisionFailed, RevisionRejected:
		return OperationFailed
	case RevisionCancelled, RevisionSuperseded:
		return OperationCancelled
	default:
		return OperationRunning
	}
}

// Stage is one of the 7 lifecycle stages (§31).
type Stage string

const (
	StageRequest     Stage = "REQUEST"
	StagePullRequest Stage = "PULL_REQUEST"
	StageCIPolicies  Stage = "CI_POLICIES"
	StageArgoCD      Stage = "ARGO_CD"
	StageCrossplane  Stage = "CROSSPLANE"
	StageCloud       Stage = "CLOUD"
	StageOperational Stage = "OPERATIONAL"
)

// Stages in journey order.
var Stages = []Stage{StageRequest, StagePullRequest, StageCIPolicies, StageArgoCD, StageCrossplane, StageCloud, StageOperational}

type StageStatus string

const (
	StagePending    StageStatus = "PENDING"
	StageInProgress StageStatus = "IN_PROGRESS"
	StageSucceeded  StageStatus = "SUCCEEDED"
	StageFailed     StageStatus = "FAILED"
	StageSkipped    StageStatus = "SKIPPED"
)

// Done reports SUCCEEDED or SKIPPED.
func (s StageStatus) Done() bool { return s == StageSucceeded || s == StageSkipped }

// ActionOwner says who must act next (objective 5).
type ActionOwner string

const (
	OwnerRequester ActionOwner = "requester"
	OwnerApprover  ActionOwner = "approver"
	OwnerPlatform  ActionOwner = "platform"
	OwnerSquad     ActionOwner = "squad"
)

// StageView is the UI-facing state of a stage (§31).
type StageView struct {
	Stage           Stage       `json:"stage"`
	Status          StageStatus `json:"status"`
	ActionOwner     ActionOwner `json:"actionOwner,omitempty"`
	Message         string      `json:"message,omitempty"`
	TechnicalDetail string      `json:"technicalDetail,omitempty"`
}

// Condition is a Kubernetes/Crossplane status condition.
type Condition struct {
	Type    string `json:"type"`
	Status  string `json:"status"` // "True" | "False" | "Unknown"
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
	// LastTransitionTime is RFC3339 as reported by the controller.
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
}

// Conditions is a list helper.
type Conditions []Condition

// Get returns the condition of the given type.
func (cs Conditions) Get(t string) (Condition, bool) {
	for _, c := range cs {
		if c.Type == t {
			return c, true
		}
	}
	return Condition{}, false
}

// IsTrue reports whether the condition t has status True.
func (cs Conditions) IsTrue(t string) bool {
	c, ok := cs.Get(t)
	return ok && c.Status == "True"
}

// FailureReason is the normalized error attached to failed revisions (§51).
type FailureReason struct {
	TechnicalCode     string `json:"technicalCode"`
	UserMessage       string `json:"userMessage"`
	TechnicalMessage  string `json:"technicalMessage,omitempty"`
	RecommendedAction string `json:"recommendedAction,omitempty"`
	Runbook           string `json:"runbook,omitempty"`
	Owner             string `json:"owner,omitempty"`
}
