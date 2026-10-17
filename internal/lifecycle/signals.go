// Package lifecycle derives resource, revision and stage state from external signals (spec §12, §31, §32).
// Everything here is pure: no I/O, no clocks other than values passed in.
package lifecycle

import (
	"strings"

	"github.com/agnaldom/crosswatc/internal/domain"
)

// ObjectSignal is the observed status of a Claim, XR or MR.
type ObjectSignal struct {
	Present            bool
	Deleting           bool
	Generation         int64
	ObservedGeneration int64
	Conditions         domain.Conditions
}

func (o ObjectSignal) ready() bool  { return o.Present && o.Conditions.IsTrue("Ready") }
func (o ObjectSignal) synced() bool { return o.Present && o.Conditions.IsTrue("Synced") }
func (o ObjectSignal) healthy() bool {
	return o.ready() && o.synced()
}

// reconcileError returns the failing condition if Synced/Ready is False with an error reason.
func (o ObjectSignal) reconcileError() (domain.Condition, bool) {
	for _, t := range []string{"Synced", "Ready"} {
		if c, ok := o.Conditions.Get(t); ok && c.Status == "False" && strings.Contains(c.Reason, "Error") {
			return c, true
		}
	}
	return domain.Condition{}, false
}

// BackstageSignal is the Scaffolder task status (§31 stage 1).
type BackstageSignal struct {
	TaskID string
	Status string // open | processing | completed | failed | cancelled
}

// CheckState aggregates required GitHub checks.
type CheckState string

const (
	ChecksNone    CheckState = ""
	ChecksPending CheckState = "pending"
	ChecksSuccess CheckState = "success"
	ChecksFailure CheckState = "failure"
)

// PRSignal is the GitHub pull request status.
type PRSignal struct {
	Found            bool
	Number           int
	URL              string
	Open             bool
	Merged           bool
	MergeCommitSHA   string
	Checks           CheckState
	Approved         bool
	ChangesRequested bool
}

// ArgoSignal is the Argo CD Application status.
type ArgoSignal struct {
	Found          bool
	SyncStatus     string // Synced | OutOfSync | Unknown
	Health         string // Healthy | Progressing | Degraded | Missing | Suspended | Unknown
	OperationPhase string // Running | Succeeded | Failed | Error | Terminating
	SyncedRevision string
	Message        string
}

// OpenRevision describes the non-terminal revision of a resource, if any.
type OpenRevision struct {
	Kind  domain.RevisionKind
	State domain.RevisionState
}

// Signals is everything the resolvers need about a resource.
type Signals struct {
	InvalidMetadata bool

	Claim ObjectSignal
	XR    ObjectSignal
	MRs   []ObjectSignal

	SecretAvailable   bool
	CatalogRegistered *bool // nil: not checked

	Backstage BackstageSignal
	PR        PRSignal
	Argo      ArgoSignal

	EverOperational bool
	OpenRevision    *OpenRevision
	CreateFailed    bool // a CREATE revision ended FAILED
	DeleteSucceeded bool // a DELETE revision ended SUCCEEDED
}

func (s Signals) mrsHealthy() bool {
	if len(s.MRs) == 0 {
		return false
	}
	for _, m := range s.MRs {
		if !m.healthy() {
			return false
		}
	}
	return true
}

func (s Signals) mrsReady() bool {
	if len(s.MRs) == 0 {
		return false
	}
	for _, m := range s.MRs {
		if !m.ready() {
			return false
		}
	}
	return true
}

// AllHealthy reports Claim, XR and every MR Ready=True and Synced=True.
func (s Signals) AllHealthy() bool {
	return s.Claim.healthy() && s.XR.healthy() && s.mrsHealthy()
}

// argoOnCommit reports whether Argo has synced the merge commit (or any commit if unknown).
func (s Signals) argoOnCommit() bool {
	if !s.Argo.Found || s.Argo.SyncStatus != "Synced" {
		return false
	}
	return s.PR.MergeCommitSHA == "" || s.Argo.SyncedRevision == "" || s.Argo.SyncedRevision == s.PR.MergeCommitSHA
}

func (s Signals) argoFailed() bool {
	return s.Argo.Found && (s.Argo.OperationPhase == "Failed" || s.Argo.OperationPhase == "Error")
}

// infraError returns the first reconcile error across XR and MRs.
func (s Signals) infraError() (domain.Condition, bool) {
	if c, ok := s.XR.reconcileError(); ok {
		return c, true
	}
	for _, m := range s.MRs {
		if c, ok := m.reconcileError(); ok {
			return c, true
		}
	}
	return domain.Condition{}, false
}
