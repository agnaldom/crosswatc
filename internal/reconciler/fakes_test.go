package reconciler

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/agnaldom/crosswatc/internal/adapters/argocd"
	"github.com/agnaldom/crosswatc/internal/adapters/github"
	"github.com/agnaldom/crosswatc/internal/domain"
	"github.com/agnaldom/crosswatc/internal/kube"
	"github.com/agnaldom/crosswatc/internal/lifecycle"
	"github.com/agnaldom/crosswatc/internal/store"
)

type memStore struct {
	mu          sync.Mutex
	resources   map[uuid.UUID]store.TrackedResource
	revisions   map[uuid.UUID]store.Revision
	operations  map[uuid.UUID]store.Operation
	projections map[uuid.UUID]store.Projection
	topologies  map[uuid.UUID]store.Topology
	events      []store.Event
	eventKeys   map[string]bool
	checkpoints map[string]error
}

func newMemStore() *memStore {
	return &memStore{
		resources: map[uuid.UUID]store.TrackedResource{}, revisions: map[uuid.UUID]store.Revision{},
		operations: map[uuid.UUID]store.Operation{}, projections: map[uuid.UUID]store.Projection{},
		topologies: map[uuid.UUID]store.Topology{}, eventKeys: map[string]bool{}, checkpoints: map[string]error{},
	}
}

func (m *memStore) UpsertResourceByClaim(_ context.Context, r store.TrackedResource) (store.TrackedResource, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, ex := range m.resources {
		if ex.ClaimNamespace == r.ClaimNamespace && ex.ClaimName == r.ClaimName && ex.ClaimKind == r.ClaimKind {
			r.ID, r.CorrelationID, r.CreatedAt = id, ex.CorrelationID, ex.CreatedAt
			r.ResourceState, r.EverOperational, r.Version = ex.ResourceState, ex.EverOperational, ex.Version+1
			m.resources[id] = r
			return r, nil
		}
	}
	r.ID, r.CreatedAt, r.ResourceState = uuid.New(), time.Now(), "UNKNOWN"
	m.resources[r.ID] = r
	return r, nil
}

func (m *memStore) UpdateResourceState(_ context.Context, id uuid.UUID, v int64, state string, everOp bool) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := m.resources[id]
	if r.Version != v {
		return 0, store.ErrConflict
	}
	r.ResourceState, r.EverOperational, r.Version = state, r.EverOperational || everOp, v+1
	m.resources[id] = r
	return r.Version, nil
}

func (m *memStore) ListResources(_ context.Context, _ store.ResourceFilter, _ store.Page) ([]store.TrackedResource, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []store.TrackedResource
	for _, r := range m.resources {
		out = append(out, r)
	}
	return out, "", nil
}

func (m *memStore) GetResourceByClaim(_ context.Context, ns, name, kind string) (store.TrackedResource, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.resources {
		if r.ClaimNamespace == ns && r.ClaimName == name && r.ClaimKind == kind {
			return r, nil
		}
	}
	return store.TrackedResource{}, store.ErrNotFound
}

func (m *memStore) ListRevisions(_ context.Context, id uuid.UUID) ([]store.Revision, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []store.Revision
	for _, r := range m.revisions {
		if r.ResourceID == id {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Rev > out[j].Rev })
	return out, nil
}

func (m *memStore) InsertRevision(_ context.Context, r store.Revision) (store.Revision, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, ex := range m.revisions {
		if ex.ResourceID == r.ResourceID && !domain.RevisionState(ex.State).Terminal() {
			return store.Revision{}, store.ErrConflict
		}
	}
	r.ID, r.StateChangedAt = uuid.New(), time.Now()
	m.revisions[r.ID] = r
	return r, nil
}

func (m *memStore) UpdateRevision(_ context.Context, r store.Revision) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ex, ok := m.revisions[r.ID]
	if !ok {
		return store.ErrNotFound
	}
	if ex.State != r.State {
		r.StateChangedAt = time.Now()
	}
	m.revisions[r.ID] = r
	return nil
}

func (m *memStore) InsertOperation(_ context.Context, o store.Operation) (store.Operation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o.ID = uuid.New()
	m.operations[o.ID] = o
	return o, nil
}

func (m *memStore) UpdateOperationsForRevision(_ context.Context, revID uuid.UUID, status string, fin *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, o := range m.operations {
		if o.RevisionID != nil && *o.RevisionID == revID {
			o.Status, o.FinishedAt = status, fin
			m.operations[id] = o
		}
	}
	return nil
}

func (m *memStore) UpsertProjection(_ context.Context, p store.Projection) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.projections[p.ResourceID] = p
	return nil
}

func (m *memStore) GetProjection(_ context.Context, id uuid.UUID) (store.Projection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.projections[id]
	if !ok {
		return p, store.ErrNotFound
	}
	return p, nil
}

func (m *memStore) UpsertTopology(_ context.Context, t store.Topology) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.topologies[t.ResourceID] = t
	return nil
}

func (m *memStore) InsertEvent(_ context.Context, e store.Event) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.eventKeys[e.DedupeKey] {
		return false, nil
	}
	m.eventKeys[e.DedupeKey] = true
	m.events = append(m.events, e)
	return true, nil
}

func (m *memStore) SaveCheckpoint(_ context.Context, src, _ string, err error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkpoints[src] = err
	return nil
}

func (m *memStore) eventTypes() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.events))
	for i, e := range m.events {
		out[i] = e.Type
	}
	return out
}

func (m *memStore) only() (store.TrackedResource, store.Revision) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var tr store.TrackedResource
	for _, r := range m.resources {
		tr = r
	}
	var rev store.Revision
	for _, r := range m.revisions {
		rev = r
	}
	return tr, rev
}

type fakeDiscovery struct {
	resources []kube.DiscoveredResource
	err       error
	xrds      []kube.XRD
	oneCalls  int
}

func (f *fakeDiscovery) DiscoverXRDs(context.Context) ([]kube.XRD, error) { return f.xrds, nil }

// DiscoverOne serves the Claim from resources; absent Claims yield a Kubernetes NotFound.
func (f *fakeDiscovery) DiscoverOne(_ context.Context, _ kube.XRD, _ []kube.XRD, _ kube.MRIndex, ns, name string) (kube.DiscoveredResource, error) {
	f.oneCalls++
	for _, d := range f.resources {
		if d.Claim.Namespace == ns && d.Claim.Name == name {
			return d, nil
		}
	}
	return kube.DiscoveredResource{}, apierrors.NewNotFound(schema.GroupResource{Resource: "claims"}, name)
}
func (f *fakeDiscovery) DiscoverMRDs(context.Context) (kube.MRIndex, error) {
	return kube.MRIndex{}, nil
}
func (f *fakeDiscovery) DiscoverResources(context.Context, []kube.XRD, kube.MRIndex) ([]kube.DiscoveredResource, error) {
	return f.resources, f.err
}

type fakePR struct {
	sig     lifecycle.PRSignal
	err     error
	lookups []github.Lookup
}

func (f *fakePR) PullRequest(_ context.Context, l github.Lookup) (lifecycle.PRSignal, error) {
	f.lookups = append(f.lookups, l)
	return f.sig, f.err
}

type fakeArgo struct{ sig lifecycle.ArgoSignal }

func (f *fakeArgo) ResolveApp(_, ann map[string]string) (argocd.AppRef, bool) {
	return argocd.AppRef{Namespace: "argocd", Name: "claims"}, ann != nil
}
func (f *fakeArgo) Status(context.Context, argocd.AppRef) (lifecycle.ArgoSignal, error) {
	return f.sig, nil
}

type fakeBackstage struct {
	sig lifecycle.BackstageSignal
	err error
}

func (f *fakeBackstage) Task(context.Context, string) (lifecycle.BackstageSignal, error) {
	return f.sig, f.err
}

var errBoom = errors.New("boom")
