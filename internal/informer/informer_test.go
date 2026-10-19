package informer

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	dynfake "k8s.io/client-go/dynamic/fake"
	mdfake "k8s.io/client-go/metadata/fake"

	"github.com/agnaldom/crosswatc/internal/contract"
	"github.com/agnaldom/crosswatc/internal/kube"
)

var (
	claimGVR = schema.GroupVersionResource{Group: "database.crosswatc.io", Version: "v1", Resource: "postgresqlinstances"}
	xrGVR    = schema.GroupVersionResource{Group: "database.crosswatc.io", Version: "v1", Resource: "xpostgresqlinstances"}
	mrGVR    = schema.GroupVersionResource{Group: "dbforpostgresql.azure.upbound.io", Version: "v1beta1", Resource: "flexibleservers"}

	claimWatch = Watch{GVR: claimGVR, Kind: "PostgreSQLInstance", Source: SourceClaim, LabelSelector: contract.ManagedSelector}
	xrWatch    = Watch{GVR: xrGVR, Kind: "XPostgreSQLInstance", Source: SourceComposite}
	mrWatch    = Watch{GVR: mrGVR, Kind: "FlexibleServer", Source: SourceManaged}
)

var pgXRD = kube.XRD{Name: "xpostgresqlinstances.database.crosswatc.io", Group: "database.crosswatc.io", Kind: "XPostgreSQLInstance",
	Plural: "xpostgresqlinstances", ClaimKind: "PostgreSQLInstance", ClaimPlural: "postgresqlinstances", Scope: "Cluster",
	Versions: []kube.XRDVersion{{Name: "v1", Served: true, Referenceable: true}}}

func object(apiVersion, kind, ns, name string, labels map[string]string, fields map[string]any) *unstructured.Unstructured {
	u := &unstructured.Unstructured{Object: map[string]any{}}
	for k, v := range fields {
		u.Object[k] = v
	}
	u.SetAPIVersion(apiVersion)
	u.SetKind(kind)
	u.SetNamespace(ns)
	u.SetName(name)
	u.SetLabels(labels)
	u.SetUID(types.UID("uid-" + name))
	return u
}

func managedClaim(name string) *unstructured.Unstructured {
	return object("database.crosswatc.io/v1", "PostgreSQLInstance", "team-a", name, map[string]string{contract.LabelManaged: "true"},
		map[string]any{"spec": map[string]any{"writeConnectionSecretToRef": map[string]any{"name": name + "-conn"}}})
}

func TestWatches(t *testing.T) {
	v2 := kube.XRD{Group: "net.crosswatc.io", Kind: "Network", Plural: "networks", Scope: "Namespaced",
		Versions: []kube.XRDVersion{{Name: "v1", Served: true}}}
	mrs := kube.MRIndex{
		{Group: mrGVR.Group, Kind: "FlexibleServer"}: {ResourceTarget: kube.ResourceTarget{Group: mrGVR.Group, Version: "v1beta1", Kind: "FlexibleServer", Plural: "flexibleservers"}, Active: true},
		{Group: mrGVR.Group, Kind: "Inactive"}:       {ResourceTarget: kube.ResourceTarget{Group: mrGVR.Group, Version: "v1beta1", Kind: "Inactive", Plural: "inactives"}},
	}
	ws := Watches([]kube.XRD{pgXRD, v2}, mrs, Options{Secrets: true, Events: true})
	keys := map[string]bool{}
	for _, w := range ws {
		keys[w.Key()] = true
	}
	for _, k := range []string{
		"xrd:apiextensions.crossplane.io/v1, Resource=compositeresourcedefinitions",
		"composition:apiextensions.crossplane.io/v1, Resource=compositions",
		"composition-revision:apiextensions.crossplane.io/v1, Resource=compositionrevisions",
		"claim:database.crosswatc.io/v1, Resource=postgresqlinstances?crosswatc.io/managed=true",
		"composite:database.crosswatc.io/v1, Resource=xpostgresqlinstances",
		"claim:net.crosswatc.io/v1, Resource=networks?crosswatc.io/managed=true", // v2 namespaced XR is user-facing
		"composite:net.crosswatc.io/v1, Resource=networks",
		"managed-resource:dbforpostgresql.azure.upbound.io/v1beta1, Resource=flexibleservers",
		"secret:/v1, Resource=secrets",
		"event:/v1, Resource=events",
	} {
		if !keys[k] {
			t.Errorf("missing watch %s", k)
		}
	}
	if len(ws) != 10 {
		t.Fatalf("inactive MR types must not be watched: %d watches", len(ws))
	}
	for _, w := range ws {
		if w.Source == SourceSecret && !w.Metadata {
			t.Fatal("secrets must use a metadata-only informer")
		}
	}
	if n := len(Watches(nil, nil, Options{})); n != 3 {
		t.Fatalf("optional watches must be off: %d", n)
	}
}

func newFakeManager(t *testing.T, objs ...runtime.Object) (*Manager, *dynfake.FakeDynamicClient, chan RawEvent) {
	t.Helper()
	scheme := runtime.NewScheme()
	dyn := dynfake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		claimGVR: "PostgreSQLInstanceList", xrGVR: "XPostgreSQLInstanceList", mrGVR: "FlexibleServerList",
	}, objs...)
	mdScheme := runtime.NewScheme()
	if err := metav1.AddMetaToScheme(mdScheme); err != nil {
		t.Fatal(err)
	}
	events := make(chan RawEvent, 16)
	m := New(dyn, mdfake.NewSimpleMetadataClient(mdScheme), nil, func(ev RawEvent) { events <- ev })
	t.Cleanup(m.Stop)
	return m, dyn, events
}

func next(t *testing.T, ch chan RawEvent) RawEvent {
	t.Helper()
	select {
	case ev := <-ch:
		return ev
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for informer event")
		return RawEvent{}
	}
}

func TestManagerEmitsChangesButNotInitialList(t *testing.T) {
	ctx := context.Background()
	m, dyn, events := newFakeManager(t, managedClaim("existing"))
	m.Ensure(ctx, []Watch{claimWatch})
	if !m.WaitForSync(ctx, 5*time.Second) {
		t.Fatal("informers did not sync")
	}
	if _, ok := m.Get(claimWatch, "team-a", "existing"); !ok {
		t.Fatal("initial object must be cached")
	}
	select {
	case ev := <-events:
		t.Fatalf("initial list must not emit events: %+v", ev)
	case <-time.After(100 * time.Millisecond):
	}

	ri := dyn.Resource(claimGVR).Namespace("team-a")
	created, err := ri.Create(ctx, managedClaim("new"), metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if ev := next(t, events); ev.Type != Add || ev.Object.GetName() != "new" || ev.Watch != claimWatch {
		t.Fatalf("got %+v", ev)
	}
	created.SetResourceVersion("2")
	created.SetAnnotations(map[string]string{"x": "y"})
	if _, err := ri.Update(ctx, created, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	if ev := next(t, events); ev.Type != Update {
		t.Fatalf("got %+v", ev)
	}
	if err := ri.Delete(ctx, "new", metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	if ev := next(t, events); ev.Type != Delete || ev.Object.GetName() != "new" {
		t.Fatalf("got %+v", ev)
	}
	if len(m.List(claimWatch)) != 1 {
		t.Fatalf("list: %d", len(m.List(claimWatch)))
	}
}

func TestManagerEnsureReconfigures(t *testing.T) {
	ctx := context.Background()
	m, _, _ := newFakeManager(t)
	m.Ensure(ctx, []Watch{claimWatch, xrWatch, {GVR: secretGVR, Kind: "Secret", Source: SourceSecret, Metadata: true}})
	if !m.WaitForSync(ctx, 5*time.Second) {
		t.Fatal("not synced")
	}
	if n := len(m.Watches()); n != 3 {
		t.Fatalf("watches %d", n)
	}
	m.Ensure(ctx, []Watch{claimWatch, mrWatch})
	ws := m.Watches()
	if len(ws) != 2 || ws[0] != claimWatch || ws[1] != mrWatch {
		t.Fatalf("got %v", ws)
	}
	if _, ok := m.Get(xrWatch, "", "x"); ok {
		t.Fatal("stopped watch must not serve objects")
	}
	m.Stop()
	if len(m.Watches()) != 0 {
		t.Fatal("stop must clear watches")
	}
}

// mapCache is an in-memory Cache for resolver tests.
type mapCache map[Watch][]metav1.Object

func (c mapCache) Watches() []Watch {
	out := make([]Watch, 0, len(c))
	for w := range c {
		out = append(out, w)
	}
	return out
}

func (c mapCache) Get(w Watch, ns, name string) (metav1.Object, bool) {
	for _, o := range c[w] {
		if o.GetNamespace() == ns && o.GetName() == name {
			return o, true
		}
	}
	return nil, false
}

func (c mapCache) List(w Watch) []metav1.Object { return c[w] }

func TestResolver(t *testing.T) {
	claim := managedClaim("pg")
	want := Target{Claim: kube.Ref{APIVersion: "database.crosswatc.io/v1", Kind: "PostgreSQLInstance", Namespace: "team-a", Name: "pg"}}
	xr := object("database.crosswatc.io/v1", "XPostgreSQLInstance", "", "pg-x1", nil, map[string]any{"spec": map[string]any{
		"claimRef": map[string]any{"apiVersion": "database.crosswatc.io/v1", "kind": "PostgreSQLInstance", "namespace": "team-a", "name": "pg"}}})
	mrByClaimLabels := object("dbforpostgresql.azure.upbound.io/v1beta1", "FlexibleServer", "", "srv",
		map[string]string{labelClaimName: "pg", labelClaimNamespace: "team-a"}, nil)
	mrByComposite := object("dbforpostgresql.azure.upbound.io/v1beta1", "FlexibleServer", "", "fw", map[string]string{labelComposite: "pg-x1"}, nil)
	orphan := object("dbforpostgresql.azure.upbound.io/v1beta1", "FlexibleServer", "", "orphan", nil, nil)
	secretWatch := Watch{GVR: secretGVR, Kind: "Secret", Source: SourceSecret, Metadata: true}
	eventWatch := Watch{GVR: eventGVR, Kind: "Event", Source: SourceEvent}
	secret := &metav1.PartialObjectMetadata{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "pg-conn"}}
	k8sEvent := func(kind, ns, name, apiVersion string) *unstructured.Unstructured {
		return object("v1", "Event", ns, "ev", nil, map[string]any{"involvedObject": map[string]any{
			"apiVersion": apiVersion, "kind": kind, "namespace": ns, "name": name}})
	}

	cache := mapCache{claimWatch: {claim}, xrWatch: {xr}, mrWatch: {mrByClaimLabels, mrByComposite, orphan}}
	r := &Resolver{Cache: cache, XRDs: []kube.XRD{pgXRD}}

	cases := map[string]struct {
		w    Watch
		o    metav1.Object
		want Target
		ok   bool
	}{
		"claim":                {claimWatch, claim, want, true},
		"xr claimRef":          {xrWatch, xr, want, true},
		"mr claim labels":      {mrWatch, mrByClaimLabels, want, true},
		"mr composite label":   {mrWatch, mrByComposite, want, true},
		"mr without owner":     {mrWatch, orphan, Target{}, false},
		"connection secret":    {secretWatch, secret, want, true},
		"unrelated secret":     {secretWatch, &metav1.PartialObjectMetadata{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "x"}}, Target{}, false},
		"event on mr":          {eventWatch, k8sEvent("FlexibleServer", "", "srv", "dbforpostgresql.azure.upbound.io/v1beta1"), want, true},
		"event on gone claim":  {eventWatch, k8sEvent("PostgreSQLInstance", "team-a", "old", "database.crosswatc.io/v1"), Target{Claim: kube.Ref{APIVersion: "database.crosswatc.io/v1", Kind: "PostgreSQLInstance", Namespace: "team-a", Name: "old"}}, true},
		"event on pod":         {eventWatch, k8sEvent("Pod", "default", "p", "v1"), Target{}, false},
		"composition":          {Watch{GVR: compositionGVR, Source: SourceComposition}, object("apiextensions.crossplane.io/v1", "Composition", "", "c", nil, nil), Target{Full: true}, true},
		"composition revision": {Watch{GVR: compositionRevisionGVR, Source: SourceCompositionRevision}, object("apiextensions.crossplane.io/v1", "CompositionRevision", "", "c-1", nil, nil), Target{Full: true}, true},
		"xrd":                  {Watch{GVR: xrdGVR, Source: SourceXRD}, object("apiextensions.crossplane.io/v1", "CompositeResourceDefinition", "", "x", nil, nil), Target{Full: true}, true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, ok := r.Resolve(RawEvent{Type: Update, Watch: tc.w, Object: tc.o})
			if ok != tc.ok || got != tc.want {
				t.Fatalf("got %+v %v, want %+v %v", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestResolverNamespacedXR(t *testing.T) {
	// Crossplane v2: a namespaced XR is user-facing; composed MRs carry crossplane.io/composite in the same namespace.
	v2 := kube.XRD{Group: "net.crosswatc.io", Kind: "Network", Plural: "networks", Scope: "Namespaced"}
	gvr := schema.GroupVersionResource{Group: "net.crosswatc.io", Version: "v1", Resource: "networks"}
	userFacing := Watch{GVR: gvr, Kind: "Network", Source: SourceClaim, LabelSelector: contract.ManagedSelector}
	composite := Watch{GVR: gvr, Kind: "Network", Source: SourceComposite}
	xr := object("net.crosswatc.io/v1", "Network", "team-b", "vnet", map[string]string{contract.LabelManaged: "true"}, nil)
	mr := object("network.azure.upbound.io/v1beta1", "VirtualNetwork", "team-b", "vnet-abc", map[string]string{labelComposite: "vnet"}, nil)
	r := &Resolver{Cache: mapCache{userFacing: {xr}, composite: {xr}, mrWatch: {mr}}, XRDs: []kube.XRD{v2}}
	want := Target{Claim: kube.Ref{APIVersion: "net.crosswatc.io/v1", Kind: "Network", Namespace: "team-b", Name: "vnet"}}
	for _, ev := range []RawEvent{{Watch: composite, Object: xr}, {Watch: mrWatch, Object: mr}} {
		if got, ok := r.Resolve(ev); !ok || got != want {
			t.Fatalf("%s: got %+v %v", ev.Watch.Key(), got, ok)
		}
	}
}
