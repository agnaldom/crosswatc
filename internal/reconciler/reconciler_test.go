package reconciler

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/agnaldom/crosswatc/internal/adapters/backstage"
	"github.com/agnaldom/crosswatc/internal/contract"
	"github.com/agnaldom/crosswatc/internal/correlation"
	"github.com/agnaldom/crosswatc/internal/domain"
	"github.com/agnaldom/crosswatc/internal/kube"
	"github.com/agnaldom/crosswatc/internal/lifecycle"
	"github.com/agnaldom/crosswatc/internal/store"
)

const reqID = "crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T"

var (
	ready    = domain.Conditions{{Type: "Ready", Status: "True"}, {Type: "Synced", Status: "True"}}
	creating = domain.Conditions{{Type: "Ready", Status: "False", Reason: "Creating"}, {Type: "Synced", Status: "True"}}
	failing  = domain.Conditions{{Type: "Ready", Status: "False"}, {Type: "Synced", Status: "False", Reason: "ReconcileError", Message: "NicInUse"}}
)

func discovered(mrConds domain.Conditions, taskID string) kube.DiscoveredResource {
	claim := kube.Object{Ref: kube.Ref{APIVersion: "database.crosswatc.io/v1", Kind: "PostgreSQLInstance", Namespace: "team-payments", Name: "pgsql-payments"},
		UID: "uid-1", Generation: 1, ObservedGeneration: 1, Conditions: ready,
		Annotations: map[string]string{contract.AnnotationRequestID: reqID},
		Spec:        map[string]any{"parameters": map[string]any{"location": "brazilsouth"}}}
	xr := kube.Object{Ref: kube.Ref{APIVersion: "database.crosswatc.io/v1", Kind: "XPostgreSQLInstance", Name: "xpg"}, Parent: claim.ID(),
		Conditions: ready, CompositionName: "postgresql-azure-v3"}
	mr := kube.Object{Ref: kube.Ref{APIVersion: "dbforpostgresql.azure.upbound.io/v1beta1", Kind: "FlexibleServer", Name: "srv"}, Parent: xr.ID(), Conditions: mrConds}
	return kube.DiscoveredResource{
		Claim: claim, XR: &xr, MRs: []kube.Object{mr}, SecretAvailable: true,
		Metadata: contract.Metadata{Owner: "payments", Environment: "dev", Provider: "azure", ResourceType: "postgresql",
			RequestID: reqID, Repository: "org/argocd", Path: "claims/dev/pg.yaml", BackstageTaskID: taskID},
	}
}

func stagesOf(t *testing.T, p store.Projection) []domain.StageView {
	t.Helper()
	var v []domain.StageView
	if err := json.Unmarshal(p.Stages, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestClusterOnlyHealthyResource(t *testing.T) {
	st := newMemStore()
	r := &Reconciler{Discovery: &fakeDiscovery{resources: []kube.DiscoveredResource{discovered(ready, "")}}, Store: st, StuckTimeout: time.Hour}
	res, err := r.RunOnce(context.Background())
	if err != nil || res.Reconciled != 1 || res.Errors != 0 {
		t.Fatalf("%+v %v", res, err)
	}
	tr, rev := st.only()
	if tr.ResourceState != "OPERATIONAL" || !tr.EverOperational || tr.CorrelationID != reqID || tr.Region != "brazilsouth" || tr.CompositionName != "postgresql-azure-v3" {
		t.Fatalf("resource: %+v", tr)
	}
	if rev.Kind != "CREATE" || rev.State != "SUCCEEDED" || rev.CompletedAt == nil {
		t.Fatalf("revision: %+v", rev)
	}
	for _, o := range st.operations {
		if o.Status != "SUCCEEDED" || o.CorrelationID != reqID || o.FinishedAt == nil {
			t.Fatalf("operation: %+v", o)
		}
	}
	want := []string{"INFRA.CLAIM.CREATED", "REQUEST.CREATED", "INFRA.MANAGED_RESOURCE.READY", "LIFECYCLE.OPERATIONAL"}
	if got := st.eventTypes(); !slices.Equal(got, want) {
		t.Fatalf("events %v", got)
	}
	stages := stagesOf(t, st.projections[tr.ID])
	if stages[0].Status != domain.StageSkipped || stages[1].Status != domain.StageSkipped || stages[6].Status != domain.StageSucceeded {
		t.Fatalf("stages %+v", stages)
	}
	if !strings.Contains(string(st.topologies[tr.ID].Nodes), "FlexibleServer") {
		t.Fatal("topology not stored")
	}
	if st.checkpoints["k8s"] != nil {
		t.Fatal("checkpoint error")
	}

	// idempotent second run: no new events, no state change
	if _, err := r.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := st.eventTypes(); len(got) != len(want) {
		t.Fatalf("second run must not emit events: %v", got)
	}
}

func TestPortalJourney(t *testing.T) {
	st := newMemStore()
	disc := &fakeDiscovery{resources: []kube.DiscoveredResource{discovered(creating, "task-1")}}
	pr := &fakePR{sig: lifecycle.PRSignal{Found: true, Number: 173, URL: "u", Open: true, Checks: lifecycle.ChecksPending}}
	argo := &fakeArgo{}
	bs := &fakeBackstage{sig: lifecycle.BackstageSignal{TaskID: "task-1", Status: "completed"}}
	r := &Reconciler{Discovery: disc, PullRequests: pr, Argo: argo, Backstage: bs, Store: st, StuckTimeout: time.Hour}
	ctx := context.Background()

	if _, err := r.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	tr, rev := st.only()
	if tr.ResourceState != "PROVISIONING" || rev.State != "VALIDATING" || rev.PRNumber == nil || *rev.PRNumber != 173 {
		t.Fatalf("after PR open: %s %+v", tr.ResourceState, rev)
	}
	if pr.lookups[0].CorrelationID != reqID || pr.lookups[0].Path != "claims/dev/pg.yaml" {
		t.Fatalf("lookup %+v", pr.lookups[0])
	}
	stages := stagesOf(t, st.projections[tr.ID])
	if stages[2].Status != domain.StageInProgress || stages[2].ActionOwner != domain.OwnerPlatform {
		t.Fatalf("CI stage %+v", stages[2])
	}

	// merged, synced, infra converged
	pr.sig = lifecycle.PRSignal{Found: true, Number: 173, Merged: true, MergeCommitSHA: "m1", Checks: lifecycle.ChecksSuccess, Approved: true}
	argo.sig = lifecycle.ArgoSignal{Found: true, SyncStatus: "Synced", Health: "Healthy", SyncedRevision: "m1"}
	disc.resources[0] = discovered(ready, "task-1")
	if _, err := r.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	tr, rev = st.only()
	if tr.ResourceState != "OPERATIONAL" || rev.State != "SUCCEEDED" || rev.CommitSHA != "m1" {
		t.Fatalf("after converge: %s %+v", tr.ResourceState, rev)
	}
	if pr.lookups[1].PRNumber != 173 {
		t.Fatal("known PR number must be reused")
	}
	for _, s := range stagesOf(t, st.projections[tr.ID]) {
		if s.Status != domain.StageSucceeded {
			t.Fatalf("stage %+v", s)
		}
	}
	var health map[string]string
	_ = json.Unmarshal(st.projections[tr.ID].Health, &health)
	if health["argocd"] != "ok" || health["cloud"] != "ok" || health["secrets"] != "ok" {
		t.Fatalf("health %v", health)
	}
}

func TestStuckRevisionFails(t *testing.T) {
	st := newMemStore()
	now := time.Now()
	r := &Reconciler{Discovery: &fakeDiscovery{resources: []kube.DiscoveredResource{discovered(creating, "")}},
		PullRequests: &fakePR{sig: lifecycle.PRSignal{Found: true, Number: 1, Merged: true, MergeCommitSHA: "m"}},
		Argo:         &fakeArgo{sig: lifecycle.ArgoSignal{Found: true, SyncStatus: "Synced", SyncedRevision: "m"}},
		Store:        st, StuckTimeout: 30 * time.Minute, Now: func() time.Time { return now }}
	ctx := context.Background()
	if _, err := r.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	_, rev := st.only()
	if rev.State != "APPLYING" {
		t.Fatalf("got %s", rev.State)
	}

	r.Discovery = &fakeDiscovery{resources: []kube.DiscoveredResource{discovered(failing, "")}}
	now = now.Add(time.Hour)
	if _, err := r.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	tr, rev := st.only()
	if rev.State != "FAILED" || !strings.Contains(string(rev.FailureReason), "ReconcileError") || tr.ResourceState != "FAILED" {
		t.Fatalf("got %s %s %s", rev.State, rev.FailureReason, tr.ResourceState)
	}
	if !slices.Contains(st.eventTypes(), "INFRA.RECONCILE.FAILED") {
		t.Fatalf("events %v", st.eventTypes())
	}
}

func TestVanishedClaim(t *testing.T) {
	st := newMemStore()
	disc := &fakeDiscovery{resources: []kube.DiscoveredResource{discovered(ready, "")}}
	r := &Reconciler{Discovery: disc, Store: st}
	ctx := context.Background()
	if _, err := r.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}

	disc.resources, disc.err = nil, errBoom // listing failure must not mark resources as gone
	if _, err := r.RunOnce(ctx); err == nil {
		t.Fatal("expected error")
	}
	if tr, _ := st.only(); tr.ResourceState != "OPERATIONAL" || st.checkpoints["k8s"] == nil {
		t.Fatalf("got %s", tr.ResourceState)
	}

	disc.err = nil
	res, err := r.RunOnce(ctx)
	if err != nil || res.Vanished != 1 {
		t.Fatalf("%+v %v", res, err)
	}
	if tr, _ := st.only(); tr.ResourceState != "UNKNOWN" {
		t.Fatalf("got %s", tr.ResourceState)
	}
	if !slices.Contains(st.eventTypes(), "INFRA.CLAIM.DELETED") {
		t.Fatal("missing INFRA.CLAIM.DELETED")
	}
}

func TestAdapterFailuresDegradeGracefully(t *testing.T) {
	st := newMemStore()
	r := &Reconciler{Discovery: &fakeDiscovery{resources: []kube.DiscoveredResource{discovered(ready, "t")}},
		PullRequests: &fakePR{err: errBoom}, Backstage: &fakeBackstage{err: backstage.ErrTaskNotFound}, Store: st}
	if res, err := r.RunOnce(context.Background()); err != nil || res.Errors != 0 {
		t.Fatalf("%+v %v", res, err)
	}
	if tr, _ := st.only(); tr.ResourceState != "OPERATIONAL" {
		t.Fatalf("got %s", tr.ResourceState)
	}
}

func TestToTrackedResource(t *testing.T) {
	d := discovered(ready, "")
	d.Metadata = contract.Metadata{RequestID: "bad"}
	tr := toTrackedResource(d)
	if !correlation.Valid(tr.CorrelationID) || tr.CorrelationID == "bad" {
		t.Fatalf("correlation id must be generated: %q", tr.CorrelationID)
	}
	if tr.Provider != "azure" || tr.Owner != "unknown" || tr.Environment != "unknown" {
		t.Fatalf("%+v", tr)
	}
	d.Metadata.Revision = 4
	if tr := toTrackedResource(d); tr.CurrentRevision == nil || *tr.CurrentRevision != 4 {
		t.Fatal("current revision")
	}
}

func TestReconcileClaim(t *testing.T) {
	st := newMemStore()
	d := discovered(creating, "")
	d.Claim.Labels = map[string]string{contract.LabelManaged: "true"}
	disc := &fakeDiscovery{
		resources: []kube.DiscoveredResource{d},
		xrds:      []kube.XRD{{Group: "database.crosswatc.io", Kind: "XPostgreSQLInstance", ClaimKind: "PostgreSQLInstance", Versions: []kube.XRDVersion{{Name: "v1", Served: true}}}},
	}
	r := &Reconciler{Discovery: disc, Store: st, StuckTimeout: time.Hour}
	ctx := context.Background()
	ref := d.Claim.Ref

	// before any full resync the trigger falls back to RunOnce
	if err := r.ReconcileClaim(ctx, ref); err != nil {
		t.Fatal(err)
	}
	if disc.oneCalls != 0 {
		t.Fatal("first trigger must run a full resync")
	}
	if tr, _ := st.only(); tr.ResourceState == "OPERATIONAL" {
		t.Fatalf("unexpected state %s", tr.ResourceState)
	}

	// MR becomes ready: targeted refresh moves the resource to OPERATIONAL
	disc.resources[0].MRs[0].Conditions = ready
	if err := r.ReconcileClaim(ctx, ref); err != nil {
		t.Fatal(err)
	}
	if tr, _ := st.only(); disc.oneCalls != 1 || tr.ResourceState != "OPERATIONAL" {
		t.Fatalf("calls=%d state=%s", disc.oneCalls, tr.ResourceState)
	}

	// unknown kinds are ignored until the next resync
	if err := r.ReconcileClaim(ctx, kube.Ref{APIVersion: "other.io/v1", Kind: "Bucket", Namespace: "x", Name: "y"}); err != nil || disc.oneCalls != 1 {
		t.Fatalf("calls=%d err=%v", disc.oneCalls, err)
	}

	// Claim deleted from the cluster → reconciled as absent
	disc.resources = nil
	if err := r.ReconcileClaim(ctx, ref); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(st.eventTypes(), "INFRA.CLAIM.DELETED") {
		t.Fatalf("events %v", st.eventTypes())
	}
	// unknown Claim gone: nothing to do
	if err := r.ReconcileClaim(ctx, kube.Ref{APIVersion: ref.APIVersion, Kind: ref.Kind, Namespace: "nope", Name: "nope"}); err != nil {
		t.Fatal(err)
	}
}
