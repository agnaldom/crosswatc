package domain

import "testing"

func TestTerminal(t *testing.T) {
	terminal := map[RevisionState]bool{RevisionSucceeded: true, RevisionFailed: true, RevisionRejected: true, RevisionCancelled: true, RevisionSuperseded: true}
	for _, s := range []RevisionState{RevisionCreated, RevisionValidating, RevisionWaitingApproval, RevisionMerged, RevisionSyncing,
		RevisionReconciling, RevisionApplying, RevisionSucceeded, RevisionFailed, RevisionRejected, RevisionCancelled, RevisionSuperseded} {
		if s.Terminal() != terminal[s] {
			t.Errorf("%s.Terminal() = %v", s, s.Terminal())
		}
	}
}

func TestOperationStatusFor(t *testing.T) {
	cases := map[RevisionState]OperationStatus{
		RevisionCreated: OperationQueued, RevisionValidating: OperationRunning, RevisionWaitingApproval: OperationWaitingApproval,
		RevisionApplying: OperationRunning, RevisionSucceeded: OperationSucceeded, RevisionRejected: OperationFailed,
		RevisionSuperseded: OperationCancelled,
	}
	for in, want := range cases {
		if got := OperationStatusFor(in); got != want {
			t.Errorf("%s → %s, want %s", in, got, want)
		}
	}
}

func TestConditions(t *testing.T) {
	cs := Conditions{{Type: "Ready", Status: "True"}, {Type: "Synced", Status: "False", Reason: "ReconcileError"}}
	if !cs.IsTrue("Ready") || cs.IsTrue("Synced") || cs.IsTrue("Missing") {
		t.Fatal("unexpected IsTrue")
	}
	if c, ok := cs.Get("Synced"); !ok || c.Reason != "ReconcileError" {
		t.Fatal("unexpected Get")
	}
}
