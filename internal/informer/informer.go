// Package informer is the Informer Manager (spec §25): it watches Claims, XRs, Managed Resources,
// Compositions, CompositionRevisions, Secret metadata and Kubernetes Events of the single control
// cluster and reports ADD/UPDATE/DELETE as raw events. Informers are read-only (spec §27).
package informer

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/agnaldom/crosswatc/internal/metrics"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/metadata/metadatainformer"
	"k8s.io/client-go/tools/cache"

	"github.com/agnaldom/crosswatc/internal/contract"
	"github.com/agnaldom/crosswatc/internal/kube"
)

// EventType is the kind of change observed (spec §25).
type EventType string

const (
	Add    EventType = "ADD"
	Update EventType = "UPDATE"
	Delete EventType = "DELETE"
)

// Sources of watched objects.
const (
	SourceClaim               = kube.RoleClaim
	SourceComposite           = kube.RoleComposite
	SourceManaged             = kube.RoleManaged
	SourceXRD                 = "xrd"
	SourceComposition         = "composition"
	SourceCompositionRevision = "composition-revision"
	SourceSecret              = "secret"
	SourceEvent               = "event"
)

var (
	xrdGVR                 = schema.GroupVersionResource{Group: "apiextensions.crossplane.io", Version: "v1", Resource: "compositeresourcedefinitions"}
	compositionGVR         = schema.GroupVersionResource{Group: "apiextensions.crossplane.io", Version: "v1", Resource: "compositions"}
	compositionRevisionGVR = schema.GroupVersionResource{Group: "apiextensions.crossplane.io", Version: "v1", Resource: "compositionrevisions"}
	secretGVR              = schema.GroupVersionResource{Version: "v1", Resource: "secrets"}
	eventGVR               = schema.GroupVersionResource{Version: "v1", Resource: "events"}
)

// Watch is one informer: a resource type, its role and an optional label selector.
type Watch struct {
	GVR           schema.GroupVersionResource
	Kind          string
	Source        string
	LabelSelector string
	Metadata      bool // metadata-only informer (Secrets: values are never read, §49)
}

// Key identifies the watch in logs.
func (w Watch) Key() string {
	k := w.Source + ":" + w.GVR.String()
	if w.LabelSelector != "" {
		k += "?" + w.LabelSelector
	}
	return k
}

// Options toggles the optional watches.
type Options struct {
	Secrets bool
	Events  bool
}

// Watches derives the informer set from discovered XRDs and MR types.
func Watches(xrds []kube.XRD, mrs kube.MRIndex, o Options) []Watch {
	ws := []Watch{
		{GVR: xrdGVR, Kind: "CompositeResourceDefinition", Source: SourceXRD},
		{GVR: compositionGVR, Kind: "Composition", Source: SourceComposition},
		{GVR: compositionRevisionGVR, Kind: "CompositionRevision", Source: SourceCompositionRevision},
	}
	for _, x := range xrds {
		if t, ok := x.UserFacingTarget(); ok && t.Version != "" {
			ws = append(ws, Watch{GVR: t.GVR(), Kind: t.Kind, Source: SourceClaim, LabelSelector: contract.ManagedSelector})
		}
		if t := x.XRTarget(); t.Version != "" {
			ws = append(ws, Watch{GVR: t.GVR(), Kind: t.Kind, Source: SourceComposite})
		}
	}
	for _, t := range mrs.Types() {
		if t.Active && t.Version != "" {
			ws = append(ws, Watch{GVR: t.GVR(), Kind: t.Kind, Source: SourceManaged})
		}
	}
	if o.Secrets {
		ws = append(ws, Watch{GVR: secretGVR, Kind: "Secret", Source: SourceSecret, Metadata: true})
	}
	if o.Events {
		ws = append(ws, Watch{GVR: eventGVR, Kind: "Event", Source: SourceEvent})
	}
	return ws
}

// RawEvent is a raw Kubernetes change, before normalization (spec §25 flow).
type RawEvent struct {
	Type   EventType
	Watch  Watch
	Object metav1.Object // *unstructured.Unstructured, or *metav1.PartialObjectMetadata for metadata watches
}

// Manager runs one informer per Watch and reconfigures them when the watch set changes.
type Manager struct {
	dyn     dynamic.Interface
	md      metadata.Interface
	log     *slog.Logger
	handler func(RawEvent)

	mu      sync.RWMutex
	running map[Watch]*running
}

type running struct {
	informer cache.SharedIndexInformer
	cancel   context.CancelFunc
}

// New builds a Manager. handler is called from informer goroutines and must not block.
func New(dyn dynamic.Interface, md metadata.Interface, log *slog.Logger, handler func(RawEvent)) *Manager {
	if log == nil {
		log = slog.Default()
	}
	return &Manager{dyn: dyn, md: md, log: log, handler: handler, running: map[Watch]*running{}}
}

// SetHandler replaces the event handler. Call before Ensure.
func (m *Manager) SetHandler(h func(RawEvent)) { m.handler = h }

// Ensure starts informers for new watches and stops the ones no longer wanted. It returns immediately;
// use HasSynced / WaitForSync for cache readiness.
func (m *Manager) Ensure(ctx context.Context, watches []Watch) {
	want := map[Watch]bool{}
	for _, w := range watches {
		if w.Metadata && m.md == nil {
			continue
		}
		want[w] = true
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for w, r := range m.running {
		if !want[w] {
			r.cancel()
			delete(m.running, w)
			m.log.Info("informer stopped", "watch", w.Key())
		}
	}
	for w := range want {
		if _, ok := m.running[w]; ok {
			continue
		}
		inf := m.newInformer(w)
		if _, err := inf.AddEventHandler(m.handlers(w)); err != nil {
			m.log.Error("informer handler", "watch", w.Key(), "err", err)
			continue
		}
		_ = inf.SetWatchErrorHandler(func(_ *cache.Reflector, err error) {
			m.log.Warn("informer watch error", "watch", w.Key(), "err", err)
			metrics.InformerRestarts.WithLabelValues(w.Key()).Inc()
		})
		ictx, cancel := context.WithCancel(ctx)
		m.running[w] = &running{informer: inf, cancel: cancel}
		go inf.Run(ictx.Done())
		m.log.Info("informer started", "watch", w.Key())
	}
}

func (m *Manager) newInformer(w Watch) cache.SharedIndexInformer {
	tweak := func(o *metav1.ListOptions) { o.LabelSelector = w.LabelSelector }
	indexers := cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}
	if w.Metadata {
		return metadatainformer.NewFilteredMetadataInformer(m.md, w.GVR, metav1.NamespaceAll, 0, indexers, tweak).Informer()
	}
	return dynamicinformer.NewFilteredDynamicInformer(m.dyn, w.GVR, metav1.NamespaceAll, 0, indexers, tweak).Informer()
}

func (m *Manager) handlers(w Watch) cache.ResourceEventHandler {
	emit := func(t EventType, obj any) {
		if d, ok := obj.(cache.DeletedFinalStateUnknown); ok {
			obj = d.Obj
		}
		o, ok := obj.(metav1.Object)
		if !ok || m.handler == nil {
			return
		}
		m.handler(RawEvent{Type: t, Watch: w, Object: o})
	}
	return cache.ResourceEventHandlerDetailedFuncs{
		AddFunc: func(obj any, initial bool) {
			if !initial { // the initial list is covered by the full resync
				emit(Add, obj)
			}
		},
		UpdateFunc: func(oldObj, newObj any) {
			o, ok1 := oldObj.(metav1.Object)
			n, ok2 := newObj.(metav1.Object)
			if ok1 && ok2 && o.GetResourceVersion() == n.GetResourceVersion() {
				return
			}
			emit(Update, newObj)
		},
		DeleteFunc: func(obj any) { emit(Delete, obj) },
	}
}

// HasSynced reports whether every running informer completed its initial list.
func (m *Manager) HasSynced() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, r := range m.running {
		if !r.informer.HasSynced() {
			return false
		}
	}
	return true
}

// WaitForSync blocks until HasSynced or ctx/timeout expires.
func (m *Manager) WaitForSync(ctx context.Context, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	t := time.NewTicker(50 * time.Millisecond)
	defer t.Stop()
	for {
		if m.HasSynced() {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-t.C:
		}
	}
}

// Stop stops every informer.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for w, r := range m.running {
		r.cancel()
		delete(m.running, w)
	}
}

// Watches returns the running watches sorted by key.
func (m *Manager) Watches() []Watch {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Watch, 0, len(m.running))
	for w := range m.running {
		out = append(out, w)
	}
	slices.SortFunc(out, func(a, b Watch) int { return strings.Compare(a.Key(), b.Key()) })
	return out
}

// Get returns a cached object.
func (m *Manager) Get(w Watch, namespace, name string) (metav1.Object, bool) {
	m.mu.RLock()
	r, ok := m.running[w]
	m.mu.RUnlock()
	if !ok {
		return nil, false
	}
	key := name
	if namespace != "" {
		key = namespace + "/" + name
	}
	obj, exists, err := r.informer.GetIndexer().GetByKey(key)
	if err != nil || !exists {
		return nil, false
	}
	o, ok := obj.(metav1.Object)
	return o, ok
}

// List returns every cached object of a watch.
func (m *Manager) List(w Watch) []metav1.Object {
	m.mu.RLock()
	r, ok := m.running[w]
	m.mu.RUnlock()
	if !ok {
		return nil
	}
	var out []metav1.Object
	for _, obj := range r.informer.GetStore().List() {
		if o, ok := obj.(metav1.Object); ok {
			out = append(out, o)
		}
	}
	return out
}
