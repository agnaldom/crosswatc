package worker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/agnaldom/crosswatc/internal/informer"
	"github.com/agnaldom/crosswatc/internal/kube"
	"github.com/agnaldom/crosswatc/internal/reconciler"
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
