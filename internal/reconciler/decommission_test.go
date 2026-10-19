package reconciler

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/agnaldom/crosswatc/internal/kube"
	"github.com/agnaldom/crosswatc/internal/lifecycle"
	"github.com/agnaldom/crosswatc/internal/store"
)

func TestDecommissionJourney(t *testing.T) {
	st := newMemStore()
	disc := &fakeDiscovery{resources: []kube.DiscoveredResource{discovered(ready, "")}}
	pr := &fakePR{}
	r := &Reconciler{Discovery: disc, PullRequests: pr, Store: st, StuckTimeout: time.Hour}
	ctx := context.Background()
	if _, err := r.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	tr, _ := st.only()
	if tr.ResourceState != "OPERATIONAL" {
		t.Fatalf("setup: %s", tr.ResourceState)
	}

	// API opened the decommission PR (rev 2, kind DELETE)
	n := 180
	now := time.Now()
	del, err := st.InsertRevision(ctx, store.Revision{ResourceID: tr.ID, Rev: 2, Kind: "DELETE", State: "VALIDATING", PRNumber: &n, StartedAt: &now})
	if err != nil {
		t.Fatal(err)
	}
	state := func() (string, string) {
		st.mu.Lock()
		defer st.mu.Unlock()
		return st.resources[tr.ID].ResourceState, st.revisions[del.ID].State
	}

	pr.sig = lifecycle.PRSignal{Found: true, Number: n, Open: true, Checks: lifecycle.ChecksSuccess}
	if _, err := r.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if rs, rv := state(); rs != "DELETE_PENDING" || rv != "WAITING_APPROVAL" {
		t.Fatalf("awaiting approval: %s %s", rs, rv)
	}

	pr.sig = lifecycle.PRSignal{Found: true, Number: n, Merged: true, MergeCommitSHA: "m", Checks: lifecycle.ChecksSuccess, Approved: true}
	d := discovered(ready, "")
	d.Claim.Deleting = true
	disc.resources = []kube.DiscoveredResource{d}
	if _, err := r.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if rs, rv := state(); rs != "DELETING" || rv != "APPLYING" {
		t.Fatalf("crossplane deleting: %s %s", rs, rv)
	}

	disc.resources = nil
	if _, err := r.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if rs, rv := state(); rs != "DELETED" || rv != "SUCCEEDED" {
		t.Fatalf("deleted: %s %s", rs, rv)
	}
	if slices.Contains(st.eventTypes(), "INFRA.CLAIM.DELETED") {
		t.Fatal("expected deletion must not be reported as an unexpected Claim loss")
	}

	// further resyncs leave DELETED resources alone
	if res, err := r.RunOnce(ctx); err != nil || res.Vanished != 0 {
		t.Fatalf("%+v %v", res, err)
	}
}
