package lifecycle

import (
	"time"

	"github.com/agnaldom/crosswatc/internal/domain"
)

var order = map[domain.RevisionState]int{
	domain.RevisionCreated:         0,
	domain.RevisionValidating:      1,
	domain.RevisionWaitingApproval: 2,
	domain.RevisionMerged:          3,
	domain.RevisionSyncing:         4,
	domain.RevisionReconciling:     5,
	domain.RevisionApplying:        6,
	domain.RevisionSucceeded:       7,
}

// RevisionResult is the outcome of evaluating a revision.
type RevisionResult struct {
	State   domain.RevisionState
	Failure *domain.FailureReason
	Stuck   bool // no progress for longer than stuckTimeout (emit OPERATION.STUCK)
}

// candidate maps raw signals to the state they indicate (§32 signal table).
func candidate(s Signals) (domain.RevisionState, *domain.FailureReason) {
	pr := s.PR
	switch {
	case !pr.Found && s.AllHealthy() && s.AllowClusterOnly && !s.DeleteRequested:
		return domain.RevisionSucceeded, nil // pre-existing/cluster-only resource already converged
	case !pr.Found:
		return domain.RevisionCreated, nil
	case !pr.Merged && !pr.Open:
		return domain.RevisionRejected, &domain.FailureReason{
			TechnicalCode: "PullRequestClosed", UserMessage: "O Pull Request foi fechado sem merge.", Owner: string(domain.OwnerRequester),
		}
	case !pr.Merged && pr.Checks == ChecksSuccess && !pr.Approved:
		return domain.RevisionWaitingApproval, nil
	case !pr.Merged && pr.Checks == ChecksFailure:
		return domain.RevisionValidating, &domain.FailureReason{
			TechnicalCode: "ChecksFailed", UserMessage: "Validações automáticas falharam.",
			RecommendedAction: "Corrija a configuração e atualize o Pull Request.", Owner: string(domain.OwnerRequester),
		}
	case !pr.Merged:
		return domain.RevisionValidating, nil
	case s.DeleteRequested && !s.Claim.Present:
		return domain.RevisionSucceeded, nil // Claim gone after merge: Crossplane finished deleting
	case s.DeleteRequested && s.Claim.Deleting:
		return domain.RevisionApplying, nil // Crossplane deleting managed resources in the cloud
	case s.argoFailed():
		return domain.RevisionFailed, &domain.FailureReason{
			TechnicalCode: "ArgoSyncFailed", UserMessage: "Falha ao sincronizar a configuração com o cluster.",
			TechnicalMessage: s.Argo.Message, Owner: string(domain.OwnerPlatform),
		}
	case s.Argo.Found && s.Argo.OperationPhase == "Running" && !s.argoOnCommit():
		return domain.RevisionSyncing, nil
	case !s.argoOnCommit():
		return domain.RevisionMerged, nil
	case s.DeleteRequested:
		return domain.RevisionReconciling, nil // synced, waiting for Argo CD to prune the Claim
	case !s.XR.synced() || s.Claim.Generation > s.Claim.ObservedGeneration:
		return domain.RevisionReconciling, nil
	case !s.Claim.ready() || !s.XR.ready() || !s.mrsReady():
		return domain.RevisionApplying, nil
	default:
		return domain.RevisionSucceeded, nil
	}
}

// EvaluateRevision computes the next state of a revision (§32).
// Transitions are monotonic: a non-terminal candidate never moves the state backwards.
// sinceProgress is the time since the revision last changed state.
func EvaluateRevision(cur domain.RevisionState, s Signals, sinceProgress, stuckTimeout time.Duration) RevisionResult {
	if cur.Terminal() {
		return RevisionResult{State: cur}
	}
	next, failure := candidate(s)
	if !next.Terminal() && order[next] < order[cur] {
		next, failure = cur, nil
	}
	res := RevisionResult{State: next, Failure: failure}
	if next != cur || stuckTimeout <= 0 || sinceProgress < stuckTimeout {
		return res
	}
	if (cur == domain.RevisionReconciling || cur == domain.RevisionApplying) && failure == nil {
		if c, ok := s.infraError(); ok {
			return RevisionResult{State: domain.RevisionFailed, Failure: &domain.FailureReason{
				TechnicalCode: c.Reason, UserMessage: "A infraestrutura não convergiu dentro do tempo esperado.",
				TechnicalMessage: c.Message, Owner: string(domain.OwnerPlatform),
			}}
		}
	}
	res.Stuck = true
	return res
}
