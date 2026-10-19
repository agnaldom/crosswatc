package reconciler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/agnaldom/crosswatc/internal/kube"
	"github.com/agnaldom/crosswatc/internal/lifecycle"
	"github.com/agnaldom/crosswatc/internal/store"
)

// operationalWith returns a store with one OPERATIONAL resource and an extra revision inserted.
func operationalWith(t *testing.T, rev store.Revision) (*memStore, *Reconciler, *fakePR, store.Revision) {
	t.Helper()
	st := newMemStore()
	pr := &fakePR{}
	r := &Reconciler{Discovery: &fakeDiscovery{resources: []kube.DiscoveredResource{discovered(ready, "")}}, PullRequests: pr, Store: st,
		StuckTimeout: time.Hour}
	if _, err := r.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	tr, _ := st.only()
	rev.ResourceID = tr.ID
	inserted, err := st.InsertRevision(context.Background(), rev)
	if err != nil {
		t.Fatal(err)
	}
	pr.lookups = nil
	return st, r, pr, inserted
}

// #73: a failed GitHub lookup must not close an in-flight revision.
func TestGitHubErrorDoesNotCloseRevision(t *testing.T) {
	n := 200
	st, r, pr, rev := operationalWith(t, store.Revision{Rev: 2, Kind: "UPDATE", State: "WAITING_APPROVAL", PRNumber: &n})
	pr.err = errors.New("github 502")
	if _, err := r.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := st.revisions[rev.ID]; got.State != "WAITING_APPROVAL" || got.CompletedAt != nil {
		t.Fatalf("revision advanced on lookup error: %+v", got)
	}
	// still evaluated normally once GitHub answers
	pr.err, pr.sig = nil, lifecycle.PRSignal{Found: true, Number: n, Open: false}
	if _, err := r.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := st.revisions[rev.ID]; got.State != "REJECTED" {
		t.Fatalf("got %s", got.State)
	}
}

// #74: before the API records the PR number, the reconciler must not adopt an older PR.
func TestPortalRevisionWithoutPRIsNotMatchedByPath(t *testing.T) {
	st, r, pr, rev := operationalWith(t, store.Revision{Rev: 2, Kind: "UPDATE", State: "CREATED"})
	pr.sig = lifecycle.PRSignal{Found: true, Number: 150, Merged: true, MergeCommitSHA: "old"} // what a path search would return
	if _, err := r.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := st.revisions[rev.ID]
	if got.State != "CREATED" || got.PRNumber != nil || got.CommitSHA != "" {
		t.Fatalf("adopted previous PR: %+v", got)
	}
	if len(pr.lookups) != 0 {
		t.Fatalf("no GitHub lookup expected without the revision's own PR number: %+v", pr.lookups)
	}
}

// #76: without a resolved Argo Application the projection must not claim "not synced with Git".
func TestAppliedStateOmitsSyncWhenArgoUnknown(t *testing.T) {
	st, _, _, _ := operationalWith(t, store.Revision{Rev: 2, Kind: "UPDATE", State: "SUCCEEDED"})
	tr, _ := st.only()
	if strings.Contains(string(st.projections[tr.ID].AppliedState), "syncedWithGit") {
		t.Fatalf("%s", st.projections[tr.ID].AppliedState)
	}
}
