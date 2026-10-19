package reconciler

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/agnaldom/crosswatc/internal/drift"
	"github.com/agnaldom/crosswatc/internal/kube"
)

type fakeGit struct {
	content string
	calls   int
}

func (f *fakeGit) GetFile(context.Context, string, string, string) ([]byte, string, error) {
	f.calls++
	return []byte(f.content), "sha", nil
}

func TestDriftDetectedAndResolved(t *testing.T) {
	st := newMemStore()
	d := discovered(ready, "")
	d.Claim.Spec = map[string]any{"parameters": map[string]any{"vmSize": "D8"}}
	git := &fakeGit{content: "metadata: {name: pgsql-payments}\nspec:\n  parameters:\n    vmSize: D4\n"}
	now := time.Now()
	r := &Reconciler{Discovery: &fakeDiscovery{resources: []kube.DiscoveredResource{d}}, Store: st, Git: git,
		GitCacheTTL: time.Minute, Now: func() time.Time { return now }}
	ctx := context.Background()

	// first run: resource becomes operational in this pass → drift evaluated with EverOperational
	if _, err := r.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	tr, _ := st.only()
	var ds drift.State
	_ = json.Unmarshal(st.projections[tr.ID].DriftState, &ds)
	if !ds.Detected || ds.Differences[0].Field != "vmSize" || ds.Differences[0].Desired != "D4" || ds.Differences[0].Observed != "D8" {
		t.Fatalf("%+v", ds)
	}
	if !slices.Contains(st.eventTypes(), "LIFECYCLE.DRIFT.DETECTED") {
		t.Fatalf("%v", st.eventTypes())
	}

	if _, err := r.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if git.calls != 1 {
		t.Fatalf("manifest must be cached, calls=%d", git.calls)
	}
	n := 0
	for _, e := range st.eventTypes() {
		if e == "LIFECYCLE.DRIFT.DETECTED" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("persisting drift must not re-emit: %v", st.eventTypes())
	}

	git.content = "metadata: {name: pgsql-payments}\nspec:\n  parameters:\n    vmSize: D8\n"
	now = now.Add(2 * time.Minute)
	if _, err := r.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(st.projections[tr.ID].DriftState, &ds)
	if ds.Detected || !slices.Contains(st.eventTypes(), "LIFECYCLE.DRIFT.RESOLVED") {
		t.Fatalf("%+v %v", ds, st.eventTypes())
	}
}
