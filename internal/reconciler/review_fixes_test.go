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

// #77: after a change succeeds, drift must compare against the new Git manifest, not the cached one.
func TestManifestCacheInvalidatedWhenRevisionFinishes(t *testing.T) {
	st := newMemStore()
	d := discovered(ready, "")
	d.Claim.Spec = map[string]any{"parameters": map[string]any{"vmSize": "D2"}}
	disc := &fakeDiscovery{resources: []kube.DiscoveredResource{d}}
	git := &fakeGit{content: "metadata: {name: pgsql-payments}\nspec:\n  parameters:\n    vmSize: D2\n"}
	pr := &fakePR{}
	argo := &fakeArgo{sig: lifecycle.ArgoSignal{Found: true, SyncStatus: "Synced", SyncedRevision: "m"}}
	now := time.Now()
	r := &Reconciler{Discovery: disc, PullRequests: pr, Argo: argo, Store: st, Git: git, GitCacheTTL: time.Hour,
		Now: func() time.Time { return now }}
	ctx := context.Background()
	if _, err := r.RunOnce(ctx); err != nil { // operational, manifest D2 cached
		t.Fatal(err)
	}
	tr, _ := st.only()
	n := 300
	if _, err := st.InsertRevision(ctx, store.Revision{ResourceID: tr.ID, Rev: 2, Kind: "UPDATE", State: "APPLYING", PRNumber: &n}); err != nil {
		t.Fatal(err)
	}
	// change merged and applied: Git and Claim both D4
	git.content = "metadata: {name: pgsql-payments}\nspec:\n  parameters:\n    vmSize: D4\n"
	d.Claim.Spec = map[string]any{"parameters": map[string]any{"vmSize": "D4"}}
	disc.resources = []kube.DiscoveredResource{d}
	pr.sig = lifecycle.PRSignal{Found: true, Number: n, Merged: true, MergeCommitSHA: "m", Checks: lifecycle.ChecksSuccess, Approved: true}
	if _, err := r.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	for _, e := range st.eventTypes() {
		if e == "LIFECYCLE.DRIFT.DETECTED" {
			t.Fatalf("false drift after a successful change: %v", st.eventTypes())
		}
	}
	if git.calls != 2 {
		t.Fatalf("manifest must be refetched after the revision finished, calls=%d", git.calls)
	}
}
