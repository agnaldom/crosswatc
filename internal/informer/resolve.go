package informer

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/agnaldom/crosswatc/internal/contract"
	"github.com/agnaldom/crosswatc/internal/kube"
)

// Crossplane labels propagated from the Claim/XR to composed resources.
const (
	labelClaimName      = "crossplane.io/claim-name"
	labelClaimNamespace = "crossplane.io/claim-namespace"
	labelComposite      = "crossplane.io/composite"
)

const maxResolveDepth = 5

// Target is what a raw event asks the reconciler to do: refresh one Claim, or resync everything.
type Target struct {
	Full  bool
	Claim kube.Ref
}

// Cache is the read side of the informer caches (satisfied by *Manager).
type Cache interface {
	Watches() []Watch
	Get(w Watch, namespace, name string) (metav1.Object, bool)
	List(w Watch) []metav1.Object
}

// Resolver maps a raw event to the user-facing Claim it affects (the "Event Normalizer" step of §25).
type Resolver struct {
	Cache Cache
	XRDs  []kube.XRD
}

// Resolve returns the target of ev; ok is false when the object belongs to no tracked Claim.
func (r *Resolver) Resolve(ev RawEvent) (Target, bool) {
	return r.resolve(ev.Watch, ev.Object, 0)
}

func (r *Resolver) resolve(w Watch, o metav1.Object, depth int) (Target, bool) {
	if o == nil || depth > maxResolveDepth {
		return Target{}, false
	}
	switch w.Source {
	case SourceXRD, SourceComposition, SourceCompositionRevision:
		return Target{Full: true}, true
	case SourceClaim:
		return Target{Claim: kube.Ref{APIVersion: w.GVR.GroupVersion().String(), Kind: w.Kind, Namespace: o.GetNamespace(), Name: o.GetName()}}, true
	case SourceComposite:
		return r.composite(w, o, depth)
	case SourceManaged:
		return r.composed(o, depth)
	case SourceSecret:
		return r.secret(o)
	case SourceEvent:
		return r.event(o, depth)
	}
	return Target{}, false
}

// composite resolves an XR: its claimRef, a user-facing namespaced XR itself, or its parent XR.
func (r *Resolver) composite(w Watch, o metav1.Object, depth int) (Target, bool) {
	if u, ok := o.(*unstructured.Unstructured); ok {
		for _, p := range [][]string{{"spec", "claimRef"}, {"spec", "crossplane", "claimRef"}} {
			m, _, _ := unstructured.NestedStringMap(u.Object, p...)
			if m["name"] != "" && m["kind"] != "" {
				return Target{Claim: kube.Ref{APIVersion: m["apiVersion"], Kind: m["kind"], Namespace: m["namespace"], Name: m["name"]}}, true
			}
		}
	}
	if o.GetLabels()[contract.LabelManaged] == "true" {
		if x, ok := r.xrdOf(w); ok && x.ClaimKind == "" && x.Scope == "Namespaced" {
			return Target{Claim: kube.Ref{APIVersion: w.GVR.GroupVersion().String(), Kind: w.Kind, Namespace: o.GetNamespace(), Name: o.GetName()}}, true
		}
	}
	return r.composed(o, depth)
}

// composed resolves an object composed by an XR (MR or nested XR) from the labels Crossplane propagates.
func (r *Resolver) composed(o metav1.Object, depth int) (Target, bool) {
	l := o.GetLabels()
	if name, ns := l[labelClaimName], l[labelClaimNamespace]; name != "" && ns != "" {
		if t, ok := r.claimByName(ns, name); ok {
			return t, true
		}
	}
	parent := l[labelComposite]
	if parent == "" {
		return Target{}, false
	}
	for _, w := range r.Cache.Watches() {
		if w.Source != SourceComposite {
			continue
		}
		// Cluster-scoped XRs are keyed by name; Crossplane v2 namespaced XRs live in the composed object's namespace.
		for _, ns := range []string{"", o.GetNamespace()} {
			p, ok := r.Cache.Get(w, ns, parent)
			if ok && (p.GetUID() == "" || p.GetUID() != o.GetUID()) {
				return r.resolve(w, p, depth+1)
			}
		}
	}
	return Target{}, false
}

// claimByName finds a tracked Claim by namespace/name across claim kinds.
func (r *Resolver) claimByName(ns, name string) (Target, bool) {
	for _, w := range r.Cache.Watches() {
		if w.Source != SourceClaim {
			continue
		}
		if c, ok := r.Cache.Get(w, ns, name); ok {
			return r.resolve(w, c, 0)
		}
	}
	return Target{}, false
}

// secret maps a connection Secret to the Claim that writes it (spec.writeConnectionSecretToRef).
func (r *Resolver) secret(o metav1.Object) (Target, bool) {
	for _, w := range r.Cache.Watches() {
		if w.Source != SourceClaim {
			continue
		}
		for _, c := range r.Cache.List(w) {
			u, ok := c.(*unstructured.Unstructured)
			if !ok {
				continue
			}
			name, _, _ := unstructured.NestedString(u.Object, "spec", "writeConnectionSecretToRef", "name")
			ns, _, _ := unstructured.NestedString(u.Object, "spec", "writeConnectionSecretToRef", "namespace")
			if ns == "" {
				ns = u.GetNamespace()
			}
			if name != "" && name == o.GetName() && ns == o.GetNamespace() {
				return r.resolve(w, c, 0)
			}
		}
	}
	return Target{}, false
}

// event maps a Kubernetes Event to its involvedObject.
func (r *Resolver) event(o metav1.Object, depth int) (Target, bool) {
	u, ok := o.(*unstructured.Unstructured)
	if !ok {
		return Target{}, false
	}
	ref, _, _ := unstructured.NestedStringMap(u.Object, "involvedObject")
	if ref == nil {
		ref, _, _ = unstructured.NestedStringMap(u.Object, "regarding") // events.k8s.io/v1
	}
	gv, err := schema.ParseGroupVersion(ref["apiVersion"])
	if err != nil || ref["name"] == "" {
		return Target{}, false
	}
	for _, w := range r.Cache.Watches() {
		if w.GVR.Group != gv.Group || w.Kind != ref["kind"] || w.Source == SourceEvent {
			continue
		}
		if obj, ok := r.Cache.Get(w, ref["namespace"], ref["name"]); ok {
			return r.resolve(w, obj, depth+1)
		}
		if w.Source == SourceClaim { // claim already gone from the cache: still refresh it
			return Target{Claim: kube.Ref{APIVersion: ref["apiVersion"], Kind: ref["kind"], Namespace: ref["namespace"], Name: ref["name"]}}, true
		}
	}
	return Target{}, false
}

func (r *Resolver) xrdOf(w Watch) (kube.XRD, bool) {
	for _, x := range r.XRDs {
		if x.Group == w.GVR.Group && x.Kind == w.Kind {
			return x, true
		}
	}
	return kube.XRD{}, false
}
