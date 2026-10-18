package drift

import (
	"slices"
	"testing"
	"time"

	"github.com/agnaldom/crosswatc/internal/domain"
	"github.com/agnaldom/crosswatc/internal/lifecycle"
)

var ok = domain.Conditions{{Type: "Ready", Status: "True"}, {Type: "Synced", Status: "True"}}

func healthy() lifecycle.Signals {
	obj := lifecycle.ObjectSignal{Present: true, Conditions: ok}
	return lifecycle.Signals{EverOperational: true, Claim: obj, XR: obj, MRs: []lifecycle.ObjectSignal{obj},
		Argo: lifecycle.ArgoSignal{Found: true, SyncStatus: "Synced"}}
}

var now = time.Date(2026, 10, 18, 10, 0, 0, 0, time.UTC)

func TestNoDrift(t *testing.T) {
	git := map[string]any{"vmSize": "D4", "disk": int64(64)}
	claim := map[string]any{"vmSize": "D4", "disk": float64(64), "defaulted": "x"}
	if s := Detect(Input{Signals: healthy(), GitParameters: git, ClaimParameters: claim, Now: now}); s.Detected {
		t.Fatalf("%+v", s)
	}
}

func TestDriftSignals(t *testing.T) {
	cases := map[string]struct {
		mut  func(*lifecycle.Signals)
		want string
	}{
		"argo": {func(s *lifecycle.Signals) { s.Argo.SyncStatus = "OutOfSync" }, SignalArgoOutOfSync},
		"xr":   {func(s *lifecycle.Signals) { s.XR.Conditions = domain.Conditions{{Type: "Synced", Status: "False"}} }, SignalXRNotSynced},
		"mr": {func(s *lifecycle.Signals) {
			s.MRs[0].Conditions = domain.Conditions{{Type: "Ready", Status: "False"}, {Type: "Synced", Status: "True"}}
		}, SignalMRNotReady},
		"cloud": {func(s *lifecycle.Signals) {
			s.MRs[0].Conditions = domain.Conditions{{Type: "Ready", Status: "True"}, {Type: "Synced", Status: "False", Reason: "ReconcileError"}}
		}, SignalCloudChangeErr},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s := healthy()
			tc.mut(&s)
			got := Detect(Input{Signals: s, Now: now})
			if !got.Detected || !slices.Contains(got.Signals, tc.want) || !got.DetectedAt.Equal(now) {
				t.Fatalf("%+v", got)
			}
		})
	}
	s := healthy()
	s.Argo.SyncStatus, s.Argo.OperationPhase = "OutOfSync", "Running"
	if Detect(Input{Signals: s, Now: now}).Detected {
		t.Fatal("argo sync in progress is not drift")
	}
}

func TestGitClaimMismatch(t *testing.T) {
	git := map[string]any{"vmSize": "D4", "network": map[string]any{"subnet": "snet-a"}, "adminPassword": "a"}
	claim := map[string]any{"vmSize": "D8", "network": map[string]any{"subnet": "snet-a"}, "adminPassword": "b"}
	prev := State{Detected: true, DetectedAt: &[]time.Time{now.Add(-time.Hour)}[0]}
	got := Detect(Input{Signals: healthy(), GitParameters: git, ClaimParameters: claim, Previous: prev, Now: now})
	if !got.Detected || got.Signals[0] != SignalGitClaim || len(got.Differences) != 2 {
		t.Fatalf("%+v", got)
	}
	if got.Differences[0].Field != "adminPassword" || got.Differences[0].Desired != "[REDACTED]" {
		t.Fatalf("sensitive values must be masked: %+v", got.Differences[0])
	}
	if d := got.Differences[1]; d.Field != "vmSize" || d.Desired != "D4" || d.Observed != "D8" {
		t.Fatalf("%+v", d)
	}
	if !got.DetectedAt.Equal(now.Add(-time.Hour)) {
		t.Fatal("detectedAt must be kept while drift persists")
	}
	missing := Compare(map[string]any{"a": 1}, map[string]any{})
	if len(missing) != 1 || missing[0].Observed != nil {
		t.Fatalf("%+v", missing)
	}
}

func TestSuppressed(t *testing.T) {
	s := healthy()
	s.Argo.SyncStatus = "OutOfSync"
	s.OpenRevision = &lifecycle.OpenRevision{Kind: domain.RevisionUpdate}
	if Detect(Input{Signals: s}).Detected {
		t.Fatal("change in flight is not drift")
	}
	s.OpenRevision, s.EverOperational = nil, false
	if Detect(Input{Signals: s}).Detected {
		t.Fatal("never-operational resources are provisioning, not drifting")
	}
}

func TestTransition(t *testing.T) {
	if d, r := Transition(State{}, State{Detected: true}); !d || r {
		t.Fatal("detected")
	}
	if d, r := Transition(State{Detected: true}, State{}); d || !r {
		t.Fatal("resolved")
	}
}
