package worker

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/agnaldom/crosswatc/internal/informer"
	"github.com/agnaldom/crosswatc/internal/kube"
	"github.com/agnaldom/crosswatc/internal/reconciler"
	"github.com/agnaldom/crosswatc/internal/store"
)

type fakeRec struct {
	mu       sync.Mutex
	full     int
	claims   []kube.Ref
	failOnce map[string]bool
	xrds     []kube.XRD
}

func (f *fakeRec) RunOnce(context.Context) (reconciler.Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.full++
	return reconciler.Result{}, nil
}

func (f *fakeRec) ReconcileClaim(_ context.Context, ref kube.Ref) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.claims = append(f.claims, ref)
	if f.failOnce[ref.Name] {
		delete(f.failOnce, ref.Name)
		return errors.New("transient")
	}
	return nil
}

func (f *fakeRec) Catalog() ([]kube.XRD, kube.MRIndex) { return f.xrds, kube.MRIndex{} }

func (f *fakeRec) snapshot() (int, []kube.Ref) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.full, append([]kube.Ref(nil), f.claims...)
}

type fakeInformers struct {
	mu      sync.Mutex
	watches []informer.Watch
	stopped bool
}

func (f *fakeInformers) Watches() []informer.Watch {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.watches
}
func (f *fakeInformers) Get(informer.Watch, string, string) (metav1.Object, bool) { return nil, false }
func (f *fakeInformers) List(informer.Watch) []metav1.Object                      { return nil }
func (f *fakeInformers) Ensure(_ context.Context, ws []informer.Watch) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.watches = ws
}
func (f *fakeInformers) HasSynced() bool { return true }
func (f *fakeInformers) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopped = true
}

func eventually(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestRunnerResyncInformersAndTargetedReconcile(t *testing.T) {
	xrd := kube.XRD{Group: "database.crosswatc.io", Kind: "XPostgreSQLInstance", ClaimKind: "PostgreSQLInstance",
		ClaimPlural: "postgresqlinstances", Plural: "xpostgresqlinstances", Versions: []kube.XRDVersion{{Name: "v1", Served: true}}}
	rec := &fakeRec{xrds: []kube.XRD{xrd}, failOnce: map[string]bool{"flaky": true}}
	inf := &fakeInformers{}
	r := &Runner{Rec: rec, Informers: inf, Resync: time.Hour, Debounce: 10 * time.Millisecond}

	if r.Ready() {
		t.Fatal("not ready before the first resync")
	}
	r.OnEvent(informer.RawEvent{}) // before Run: ignored, no panic

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.Run(ctx) }()

	eventually(t, func() bool { full, _ := rec.snapshot(); return full == 1 && r.Ready() })
	if len(inf.Watches()) == 0 {
		t.Fatal("informers not configured after resync")
	}

	claimWatch := informer.Watch{GVR: schema.GroupVersionResource{Group: "database.crosswatc.io", Version: "v1", Resource: "postgresqlinstances"},
		Kind: "PostgreSQLInstance", Source: informer.SourceClaim}
	claim := func(name string) *unstructured.Unstructured {
		u := &unstructured.Unstructured{}
		u.SetNamespace("team-a")
		u.SetName(name)
		return u
	}
	// a burst of events for the same Claim is coalesced
	for range 5 {
		r.OnEvent(informer.RawEvent{Type: informer.Update, Watch: claimWatch, Object: claim("pg")})
	}
	r.OnEvent(informer.RawEvent{Type: informer.Add, Watch: claimWatch, Object: claim("flaky")})

	count := func(name string) int {
		_, claims := rec.snapshot()
		n := 0
		for _, c := range claims {
			if c.Name == name {
				n++
			}
		}
		return n
	}
	eventually(t, func() bool { return count("pg") >= 1 && count("flaky") == 2 }) // failure retried with backoff
	if n := count("pg"); n != 1 {
		t.Fatalf("burst not coalesced: %d reconciles", n)
	}
	_, claims := rec.snapshot()
	for _, c := range claims {
		if c.Kind != "PostgreSQLInstance" || c.Namespace != "team-a" || c.APIVersion != "database.crosswatc.io/v1" {
			t.Fatalf("ref %+v", c)
		}
	}

	r.Enqueue(informer.Target{Full: true})
	eventually(t, func() bool { full, _ := rec.snapshot(); return full == 2 })

	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if !inf.stopped || r.Ready() {
		t.Fatal("informers must stop and readiness drop when leadership ends")
	}
	r.Enqueue(informer.Target{Full: true}) // after Run: no-op
}

type fakeInbox struct {
	mu        sync.Mutex
	items     []store.InboxItem
	results   map[string]error
	resources []store.TrackedResource
	listErr   error
}

func (f *fakeInbox) ProcessInbox(_ context.Context, limit, _ int, fn func(store.InboxItem) error) (int, error) {
	f.mu.Lock()
	items := f.items
	f.items = nil
	f.mu.Unlock()
	for _, it := range items {
		err := fn(it)
		f.mu.Lock()
		f.results[it.DeliveryID] = err
		f.mu.Unlock()
	}
	return len(items), nil
}

func (f *fakeInbox) ListResources(_ context.Context, flt store.ResourceFilter, _ store.Page) ([]store.TrackedResource, string, error) {
	if f.listErr != nil {
		return nil, "", f.listErr
	}
	var out []store.TrackedResource
	for _, r := range f.resources {
		if (len(flt.CorrelationIDs) > 0 && slices.Contains(flt.CorrelationIDs, r.CorrelationID)) ||
			(flt.Repository != "" && flt.Repository == r.Repository) {
			out = append(out, r)
		}
	}
	return out, "", nil
}

func (f *fakeInbox) result(id string) (error, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	err, ok := f.results[id]
	return err, ok
}

func TestRunnerRoutesWebhooksToClaims(t *testing.T) {
	const reqA = "crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T"
	res := func(name, corr string) store.TrackedResource {
		return store.TrackedResource{Name: name, CorrelationID: corr, Repository: "org/gitops", ClaimAPIVersion: "database.crosswatc.io/v1",
			ClaimKind: "PostgreSQLInstance", ClaimNamespace: "team-a", ClaimName: name}
	}
	inbox := &fakeInbox{results: map[string]error{}, resources: []store.TrackedResource{res("pg-a", reqA), res("pg-b", "other")}}
	inbox.items = []store.InboxItem{
		{Source: "github", DeliveryID: "by-corr", EventType: "pull_request", ReceivedAt: time.Now(),
			Payload: []byte(`{"action":"opened","repository":{"full_name":"org/gitops"},"pull_request":{"number":1,"body":"` + reqA + `"}}`)},
		{Source: "github", DeliveryID: "by-repo", EventType: "push", ReceivedAt: time.Now(),
			Payload: []byte(`{"ref":"refs/heads/main","repository":{"full_name":"org/gitops"}}`)},
		{Source: "argocd", DeliveryID: "argo", EventType: "sync-succeeded", ReceivedAt: time.Now(),
			Payload: []byte(`{"eventId":"e","app":"claims","event":"sync-succeeded"}`)},
		{Source: "github", DeliveryID: "malformed", EventType: "push", Payload: []byte(`nope`)},
		{Source: "gitlab", DeliveryID: "unknown-source", EventType: "push", Payload: []byte(`{}`)},
	}
	rec := &fakeRec{failOnce: map[string]bool{}}
	r := &Runner{Rec: rec, Resync: time.Hour, Debounce: 5 * time.Millisecond, Inbox: inbox, InboxInterval: 10 * time.Millisecond}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = r.Run(ctx) }()

	eventually(t, func() bool { _, ok := inbox.result("unknown-source"); return ok })
	for _, id := range []string{"malformed", "unknown-source"} {
		if err, _ := inbox.result(id); !errors.Is(err, store.ErrPermanent) {
			t.Fatalf("%s must be dead-lettered, got %v", id, err)
		}
	}
	for _, id := range []string{"by-corr", "by-repo", "argo"} {
		if err, _ := inbox.result(id); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
	}
	// pg-a via correlation id, pg-a + pg-b via repository, and a full resync for Argo
	eventually(t, func() bool {
		full, claims := rec.snapshot()
		names := map[string]bool{}
		for _, c := range claims {
			names[c.Name] = true
		}
		return names["pg-a"] && names["pg-b"] && full >= 2
	})

	// lookup failures are returned so the delivery is retried
	inbox.listErr = errors.New("db down")
	if err := r.handleDelivery(ctx, inbox.resourcesItem()); err == nil || errors.Is(err, store.ErrPermanent) {
		t.Fatalf("got %v", err)
	}
}

func (f *fakeInbox) resourcesItem() store.InboxItem {
	return store.InboxItem{Source: "github", DeliveryID: "x", EventType: "push", Payload: []byte(`{"repository":{"full_name":"org/gitops"}}`)}
}
