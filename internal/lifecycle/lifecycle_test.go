package lifecycle

import (
	"testing"
	"time"

	"github.com/agnaldom/crosswatc/internal/domain"
)

func cond(t, status, reason string) domain.Condition {
	return domain.Condition{Type: t, Status: status, Reason: reason}
}

func healthy() ObjectSignal {
	return ObjectSignal{Present: true, Generation: 1, ObservedGeneration: 1,
		Conditions: domain.Conditions{cond("Ready", "True", "Available"), cond("Synced", "True", "ReconcileSuccess")}}
}

func notReady() ObjectSignal {
	return ObjectSignal{Present: true, Conditions: domain.Conditions{cond("Ready", "False", "Creating"), cond("Synced", "True", "")}}
}

func failing() ObjectSignal {
	return ObjectSignal{Present: true, Conditions: domain.Conditions{cond("Ready", "False", ""), cond("Synced", "False", "ReconcileError")}}
}

// happy is a fully provisioned resource created through the Portal and a merged PR.
func happy() Signals {
	return Signals{
		Claim: healthy(), XR: healthy(), MRs: []ObjectSignal{healthy(), healthy()},
		SecretAvailable: true,
		Backstage:       BackstageSignal{TaskID: "t1", Status: "completed"},
		PR:              PRSignal{Found: true, Number: 173, Merged: true, MergeCommitSHA: "abc", Checks: ChecksSuccess, Approved: true},
		Argo:            ArgoSignal{Found: true, SyncStatus: "Synced", Health: "Healthy", OperationPhase: "Succeeded", SyncedRevision: "abc"},
	}
}

func TestDeriveResourceState(t *testing.T) {
	upd := &OpenRevision{Kind: domain.RevisionUpdate, State: domain.RevisionValidating}
	cases := []struct {
		name string
		mut  func(*Signals)
		want domain.ResourceState
	}{
		{"operational", func(*Signals) {}, domain.ResourceOperational},
		{"deleting wins", func(s *Signals) { s.Claim.Deleting = true; s.OpenRevision = upd }, domain.ResourceDeleting},
		{"delete pending", func(s *Signals) { s.OpenRevision = &OpenRevision{Kind: domain.RevisionDelete} }, domain.ResourceDeletePending},
		{"deleted", func(s *Signals) { s.Claim = ObjectSignal{}; s.DeleteSucceeded = true }, domain.ResourceDeleted},
		{"claim missing", func(s *Signals) { s.Claim = ObjectSignal{} }, domain.ResourceUnknown},
		{"invalid metadata", func(s *Signals) { s.InvalidMetadata = true }, domain.ResourceUnknown},
		{"create failed", func(s *Signals) { s.MRs[0] = failing(); s.CreateFailed = true }, domain.ResourceFailed},
		{"create failed but was operational", func(s *Signals) { s.MRs[0] = failing(); s.CreateFailed = true; s.EverOperational = true }, domain.ResourceDegraded},
		{"provisioning", func(s *Signals) { s.MRs[0] = notReady() }, domain.ResourceProvisioning},
		{"no MRs yet", func(s *Signals) { s.MRs = nil }, domain.ResourceProvisioning},
		{"degraded", func(s *Signals) { s.MRs[0] = notReady(); s.EverOperational = true }, domain.ResourceDegraded},
		{"updating healthy", func(s *Signals) { s.OpenRevision = upd; s.EverOperational = true }, domain.ResourceUpdating},
		{"updating unhealthy", func(s *Signals) { s.OpenRevision = upd; s.EverOperational = true; s.MRs[0] = notReady() }, domain.ResourceUpdating},
		{"upgrade counts as updating", func(s *Signals) { s.OpenRevision = &OpenRevision{Kind: domain.RevisionPlatformUpgrade} }, domain.ResourceUpdating},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := happy()
			tc.mut(&s)
			if got := DeriveResourceState(s); got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func statuses(v []domain.StageView) []domain.StageStatus {
	out := make([]domain.StageStatus, len(v))
	for i, s := range v {
		out[i] = s.Status
	}
	return out
}

const (
	P = domain.StagePending
	I = domain.StageInProgress
	S = domain.StageSucceeded
	F = domain.StageFailed
	K = domain.StageSkipped
)

func TestResolveStages(t *testing.T) {
	cases := []struct {
		name  string
		mut   func(*Signals)
		want  []domain.StageStatus
		owner domain.ActionOwner // owner of the first non-done stage
	}{
		{"all done", func(*Signals) {}, []domain.StageStatus{S, S, S, S, S, S, S}, ""},
		{"no backstage task", func(s *Signals) { s.Backstage = BackstageSignal{} }, []domain.StageStatus{K, S, S, S, S, S, S}, ""},
		{"scaffolder running", func(s *Signals) { s.Backstage.Status = "processing" }, []domain.StageStatus{I, P, P, P, P, P, P}, domain.OwnerPlatform},
		{"scaffolder failed", func(s *Signals) { s.Backstage.Status = "failed" }, []domain.StageStatus{F, P, P, P, P, P, P}, domain.OwnerRequester},
		{"waiting PR", func(s *Signals) {
			s.PR = PRSignal{}
			s.Claim, s.Argo = ObjectSignal{}, ArgoSignal{}
		}, []domain.StageStatus{S, I, P, P, P, P, P}, domain.OwnerPlatform},
		{"cluster only skips PR and CI", func(s *Signals) { s.PR = PRSignal{} }, []domain.StageStatus{S, K, K, S, S, S, S}, ""},
		{"PR closed", func(s *Signals) { s.PR.Merged, s.PR.Open = false, false }, []domain.StageStatus{S, F, P, P, P, P, P}, domain.OwnerRequester},
		{"checks running", func(s *Signals) { s.PR.Merged, s.PR.Open, s.PR.Checks = false, true, ChecksPending }, []domain.StageStatus{S, S, I, P, P, P, P}, domain.OwnerPlatform},
		{"checks failed", func(s *Signals) { s.PR.Merged, s.PR.Open, s.PR.Checks = false, true, ChecksFailure }, []domain.StageStatus{S, S, F, P, P, P, P}, domain.OwnerRequester},
		{"waiting approval", func(s *Signals) { s.PR.Merged, s.PR.Open, s.PR.Approved = false, true, false }, []domain.StageStatus{S, S, I, P, P, P, P}, domain.OwnerApprover},
		{"changes requested", func(s *Signals) { s.PR.Merged, s.PR.Open, s.PR.ChangesRequested = false, true, true }, []domain.StageStatus{S, S, I, P, P, P, P}, domain.OwnerRequester},
		{"argo syncing old commit", func(s *Signals) { s.Argo.SyncedRevision = "old" }, []domain.StageStatus{S, S, S, I, P, P, P}, domain.OwnerPlatform},
		{"argo failed", func(s *Signals) { s.Argo.OperationPhase = "Failed" }, []domain.StageStatus{S, S, S, F, P, P, P}, domain.OwnerPlatform},
		{"argo unknown but claim present", func(s *Signals) { s.Argo = ArgoSignal{} }, []domain.StageStatus{S, S, S, K, S, S, S}, ""},
		{"xr composing", func(s *Signals) { s.XR = notReady(); s.XR.Conditions[1].Status = "False" }, []domain.StageStatus{S, S, S, S, I, P, P}, domain.OwnerPlatform},
		{"xr error", func(s *Signals) { s.XR = failing() }, []domain.StageStatus{S, S, S, S, F, P, P}, domain.OwnerPlatform},
		{"cloud provisioning", func(s *Signals) { s.MRs[1] = notReady() }, []domain.StageStatus{S, S, S, S, S, I, P}, domain.OwnerPlatform},
		{"cloud error", func(s *Signals) { s.MRs[1] = failing() }, []domain.StageStatus{S, S, S, S, S, F, P}, domain.OwnerPlatform},
		{"secret missing", func(s *Signals) { s.SecretAvailable = false }, []domain.StageStatus{S, S, S, S, S, S, I}, domain.OwnerPlatform},
		{"catalog missing", func(s *Signals) { f := false; s.CatalogRegistered = &f }, []domain.StageStatus{S, S, S, S, S, S, I}, domain.OwnerPlatform},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := happy()
			tc.mut(&s)
			got := ResolveStages(s)
			if len(got) != 7 {
				t.Fatalf("want 7 stages, got %d", len(got))
			}
			for i, v := range got {
				if v.Stage != domain.Stages[i] {
					t.Fatalf("stage %d = %s", i, v.Stage)
				}
			}
			gs := statuses(got)
			for i := range gs {
				if gs[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", gs, tc.want)
				}
			}
			for _, v := range got {
				if !v.Status.Done() {
					if v.ActionOwner != tc.owner {
						t.Fatalf("owner of %s = %q, want %q", v.Stage, v.ActionOwner, tc.owner)
					}
					break
				}
			}
		})
	}
}

func TestEvaluateRevisionSignals(t *testing.T) {
	cases := []struct {
		name    string
		mut     func(*Signals)
		want    domain.RevisionState
		failure string
	}{
		{"no PR", func(s *Signals) { s.PR = PRSignal{} }, domain.RevisionCreated, ""},
		{"validating", func(s *Signals) { s.PR.Merged, s.PR.Open, s.PR.Checks = false, true, ChecksPending }, domain.RevisionValidating, ""},
		{"checks failed keeps validating with reason", func(s *Signals) { s.PR.Merged, s.PR.Open, s.PR.Checks = false, true, ChecksFailure }, domain.RevisionValidating, "ChecksFailed"},
		{"waiting approval", func(s *Signals) { s.PR.Merged, s.PR.Open, s.PR.Approved = false, true, false }, domain.RevisionWaitingApproval, ""},
		{"rejected", func(s *Signals) { s.PR.Merged, s.PR.Open = false, false }, domain.RevisionRejected, "PullRequestClosed"},
		{"merged, argo not yet", func(s *Signals) { s.Argo = ArgoSignal{Found: true, SyncStatus: "OutOfSync"} }, domain.RevisionMerged, ""},
		{"syncing", func(s *Signals) { s.Argo.OperationPhase, s.Argo.SyncedRevision = "Running", "old" }, domain.RevisionSyncing, ""},
		{"sync failed", func(s *Signals) { s.Argo.OperationPhase = "Error" }, domain.RevisionFailed, "ArgoSyncFailed"},
		{"reconciling generation", func(s *Signals) { s.Claim.Generation = 2 }, domain.RevisionReconciling, ""},
		{"reconciling xr", func(s *Signals) { s.XR.Conditions[1].Status = "False" }, domain.RevisionReconciling, ""},
		{"applying", func(s *Signals) { s.MRs[0] = notReady() }, domain.RevisionApplying, ""},
		{"succeeded", func(*Signals) {}, domain.RevisionSucceeded, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := happy()
			tc.mut(&s)
			got := EvaluateRevision(domain.RevisionCreated, s, 0, time.Hour)
			if got.State != tc.want {
				t.Fatalf("got %s, want %s", got.State, tc.want)
			}
			code := ""
			if got.Failure != nil {
				code = got.Failure.TechnicalCode
			}
			if code != tc.failure {
				t.Fatalf("failure %q, want %q", code, tc.failure)
			}
		})
	}
}

func TestEvaluateRevisionMonotonicAndTerminal(t *testing.T) {
	s := happy()
	s.Argo.SyncedRevision = "old" // would indicate MERGED
	if got := EvaluateRevision(domain.RevisionApplying, s, 0, time.Hour); got.State != domain.RevisionApplying {
		t.Fatalf("must not regress, got %s", got.State)
	}
	s = happy()
	s.PR.Merged, s.PR.Open = false, false // closed
	if got := EvaluateRevision(domain.RevisionSucceeded, s, 0, time.Hour); got.State != domain.RevisionSucceeded {
		t.Fatalf("terminal must stick, got %s", got.State)
	}
	if got := EvaluateRevision(domain.RevisionWaitingApproval, s, 0, time.Hour); got.State != domain.RevisionRejected {
		t.Fatalf("terminal candidate must apply, got %s", got.State)
	}
}

func TestEvaluateRevisionStuck(t *testing.T) {
	s := happy()
	s.MRs[0] = failing()
	if got := EvaluateRevision(domain.RevisionApplying, s, 10*time.Minute, 30*time.Minute); got.State != domain.RevisionApplying || got.Stuck {
		t.Fatalf("before timeout: %+v", got)
	}
	got := EvaluateRevision(domain.RevisionApplying, s, 31*time.Minute, 30*time.Minute)
	if got.State != domain.RevisionFailed || got.Failure == nil || got.Failure.TechnicalCode != "ReconcileError" {
		t.Fatalf("after timeout with error: %+v", got)
	}

	s = happy()
	s.MRs[0] = notReady() // slow but no error
	got = EvaluateRevision(domain.RevisionApplying, s, 31*time.Minute, 30*time.Minute)
	if got.State != domain.RevisionApplying || !got.Stuck {
		t.Fatalf("slow without error must flag stuck: %+v", got)
	}

	s = happy()
	s.PR.Merged, s.PR.Open, s.PR.Approved = false, true, false
	got = EvaluateRevision(domain.RevisionWaitingApproval, s, 2*time.Hour, 30*time.Minute)
	if got.State != domain.RevisionWaitingApproval || !got.Stuck {
		t.Fatalf("waiting approval long time must flag stuck: %+v", got)
	}
}
