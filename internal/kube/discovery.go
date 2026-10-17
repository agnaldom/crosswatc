package kube

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/agnaldom/crosswatc/internal/contract"
	"github.com/agnaldom/crosswatc/internal/domain"
)

// Roles of discovered objects.
const (
	RoleClaim     = "claim"
	RoleComposite = "composite"
	RoleManaged   = "managed-resource"
)

// maxCompositeDepth bounds nested composition walking.
const maxCompositeDepth = 5

// Ref points to a Kubernetes object.
type Ref struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace,omitempty"`
}

// ID is a stable node identifier: kind/namespace/name (namespace omitted when empty).
func (r Ref) ID() string {
	if r.Namespace == "" {
		return r.Kind + "/" + r.Name
	}
	return r.Kind + "/" + r.Namespace + "/" + r.Name
}

// MissingRef is a reference that could not be resolved, with the ID of the object that referenced it.
type MissingRef struct {
	Ref
	Parent string
}

// Object is the normalized view of a Claim, XR or MR.
type Object struct {
	Ref
	Role                    string
	Parent                  string // ID of the composing object ("" for the user-facing root)
	UID                     string
	Labels                  map[string]string
	Annotations             map[string]string
	Generation              int64
	ObservedGeneration      int64
	Deleting                bool
	Conditions              domain.Conditions
	CompositionName         string
	CompositionRevision     string
	CompositionResourceName string
	ExternalName            string
	Spec                    map[string]any
	resourceRefs            []Ref
}

// Ready reports condition Ready=True.
func (o Object) Ready() bool { return o.Conditions.IsTrue("Ready") }

// Synced reports condition Synced=True.
func (o Object) Synced() bool { return o.Conditions.IsTrue("Synced") }

// DiscoveredResource is a user-facing Claim (or namespaced XR) and everything it composes.
type DiscoveredResource struct {
	XRD             XRD
	Claim           Object
	XR              *Object  // nil while the XR does not exist yet
	Composites      []Object // nested XRs
	MRs             []Object
	Missing         []MissingRef // referenced but not found
	SecretAvailable bool
	Metadata        contract.Metadata
	MetadataErrors  []contract.FieldError
	Errors          []string // non-fatal lookup errors
}

// DiscoverResources finds every Claim labelled crosswatc.io/managed=true and resolves Claim → XR → MRs (§48).
// Per-object failures are recorded on the resource; listing failures are returned joined.
func (c *Client) DiscoverResources(ctx context.Context, xrds []XRD, mrs MRIndex) ([]DiscoveredResource, error) {
	composites := map[schema.GroupKind]XRD{}
	for _, x := range xrds {
		composites[schema.GroupKind{Group: x.Group, Kind: x.Kind}] = x
	}
	var (
		out  []DiscoveredResource
		errs []error
	)
	for _, x := range xrds {
		target, ok := x.UserFacingTarget()
		if !ok || target.Version == "" {
			continue
		}
		err := c.ForEachPage(ctx, target.GVR(), ListOptions{LabelSelector: contract.ManagedSelector}, func(items []unstructured.Unstructured) error {
			for i := range items {
				out = append(out, c.resolve(ctx, x, &items[i], composites, mrs))
			}
			return nil
		})
		if err != nil {
			errs = append(errs, err)
		}
	}
	return out, errors.Join(errs...)
}

// DiscoverOne resolves a single user-facing object (used after informer/webhook triggers).
func (c *Client) DiscoverOne(ctx context.Context, x XRD, xrds []XRD, mrs MRIndex, namespace, name string) (DiscoveredResource, error) {
	target, ok := x.UserFacingTarget()
	if !ok {
		return DiscoveredResource{}, fmt.Errorf("kube: XRD %s has no user-facing type", x.Name)
	}
	u, err := c.Get(ctx, target.GVR(), namespace, name)
	if err != nil {
		return DiscoveredResource{}, err
	}
	composites := map[schema.GroupKind]XRD{}
	for _, xx := range xrds {
		composites[schema.GroupKind{Group: xx.Group, Kind: xx.Kind}] = xx
	}
	return c.resolve(ctx, x, u, composites, mrs), nil
}

func (c *Client) resolve(ctx context.Context, x XRD, u *unstructured.Unstructured, composites map[schema.GroupKind]XRD, mrs MRIndex) DiscoveredResource {
	_, hasClaim := x.ClaimTarget()
	d := DiscoveredResource{XRD: x}
	d.Claim = toObject(u, RoleClaim, "")
	d.Metadata, d.MetadataErrors = contract.Parse(u.GetLabels(), u.GetAnnotations())

	if hasClaim {
		if ref, ok := claimResourceRef(u); ok {
			xr, err := c.getRef(ctx, ref, x.XRTarget().Plural, false)
			switch {
			case err == nil:
				o := toObject(xr, RoleComposite, d.Claim.ID())
				d.XR = &o
			case IsNotFound(err):
				d.Missing = append(d.Missing, MissingRef{ref, d.Claim.ID()})
			default:
				d.Errors = append(d.Errors, err.Error())
			}
		}
	} else {
		o := d.Claim
		o.Role = RoleComposite
		d.XR = &o
	}

	if d.XR != nil {
		c.walk(ctx, &d, *d.XR, composites, mrs, 0)
	}
	d.SecretAvailable = c.secretAvailable(ctx, &d, u)
	return d
}

// walk follows resourceRefs of a composite, recursing into nested XRs.
func (c *Client) walk(ctx context.Context, d *DiscoveredResource, parent Object, composites map[schema.GroupKind]XRD, mrs MRIndex, depth int) {
	if depth >= maxCompositeDepth {
		d.Errors = append(d.Errors, fmt.Sprintf("composition depth limit reached at %s", parent.ID()))
		return
	}
	for _, ref := range parent.resourceRefs {
		gv, err := schema.ParseGroupVersion(ref.APIVersion)
		if err != nil {
			d.Errors = append(d.Errors, fmt.Sprintf("invalid apiVersion %q in %s", ref.APIVersion, parent.ID()))
			continue
		}
		if ref.Namespace == "" && parent.Namespace != "" {
			ref.Namespace = parent.Namespace // Crossplane v2: composed resources live in the XR namespace
		}
		if nested, ok := composites[schema.GroupKind{Group: gv.Group, Kind: ref.Kind}]; ok {
			u, err := c.getRef(ctx, ref, nested.Plural, nested.Scope == "Namespaced")
			if c.recordLookup(d, ref, parent.ID(), err) {
				o := toObject(u, RoleComposite, parent.ID())
				d.Composites = append(d.Composites, o)
				c.walk(ctx, d, o, composites, mrs, depth+1)
			}
			continue
		}
		t, ok := mrs.Lookup(ref.APIVersion, ref.Kind)
		if !ok {
			d.Errors = append(d.Errors, fmt.Sprintf("unknown managed resource type %s %s", ref.APIVersion, ref.Kind))
			continue
		}
		u, err := c.getRef(ctx, ref, t.Plural, t.Namespaced)
		if c.recordLookup(d, ref, parent.ID(), err) {
			d.MRs = append(d.MRs, toObject(u, RoleManaged, parent.ID()))
		}
	}
}

func (c *Client) recordLookup(d *DiscoveredResource, ref Ref, parent string, err error) bool {
	switch {
	case err == nil:
		return true
	case IsNotFound(err):
		d.Missing = append(d.Missing, MissingRef{ref, parent})
	default:
		d.Errors = append(d.Errors, err.Error())
	}
	return false
}

func (c *Client) getRef(ctx context.Context, ref Ref, plural string, namespaced bool) (*unstructured.Unstructured, error) {
	gv, err := schema.ParseGroupVersion(ref.APIVersion)
	if err != nil {
		return nil, err
	}
	ns := ""
	if namespaced {
		ns = ref.Namespace
	}
	return c.Get(ctx, gv.WithResource(plural), ns, ref.Name)
}

// secretAvailable checks spec.writeConnectionSecretToRef by metadata only.
// Resources that publish no connection secret count as available.
func (c *Client) secretAvailable(ctx context.Context, d *DiscoveredResource, u *unstructured.Unstructured) bool {
	name := str(u.Object, "spec", "writeConnectionSecretToRef", "name")
	if name == "" || c.Metadata == nil {
		return name == ""
	}
	ns := str(u.Object, "spec", "writeConnectionSecretToRef", "namespace")
	if ns == "" {
		ns = u.GetNamespace()
	}
	ok, err := c.SecretExists(ctx, ns, name)
	if err != nil {
		d.Errors = append(d.Errors, err.Error())
	}
	return ok
}

func claimResourceRef(u *unstructured.Unstructured) (Ref, bool) {
	for _, path := range [][]string{{"spec", "resourceRef"}, {"spec", "crossplane", "resourceRef"}} {
		m := obj(u.Object, path...)
		if r := toRef(m); r.Name != "" {
			return r, true
		}
	}
	return Ref{}, false
}

func toRef(m map[string]any) Ref {
	return Ref{APIVersion: str(m, "apiVersion"), Kind: str(m, "kind"), Name: str(m, "name"), Namespace: str(m, "namespace")}
}

func toObject(u *unstructured.Unstructured, role, parent string) Object {
	o := Object{
		Ref:                     Ref{APIVersion: u.GetAPIVersion(), Kind: u.GetKind(), Name: u.GetName(), Namespace: u.GetNamespace()},
		Role:                    role,
		Parent:                  parent,
		UID:                     string(u.GetUID()),
		Labels:                  u.GetLabels(),
		Annotations:             u.GetAnnotations(),
		Generation:              u.GetGeneration(),
		Deleting:                u.GetDeletionTimestamp() != nil,
		Conditions:              Conditions(u),
		CompositionResourceName: u.GetAnnotations()["crossplane.io/composition-resource-name"],
		ExternalName:            u.GetAnnotations()["crossplane.io/external-name"],
		Spec:                    obj(u.Object, "spec"),
	}
	o.ObservedGeneration, _, _ = unstructured.NestedInt64(u.Object, "status", "observedGeneration")
	for _, p := range [][]string{{"spec"}, {"spec", "crossplane"}} {
		if o.CompositionName == "" {
			o.CompositionName = str(u.Object, append(p, "compositionRef", "name")...)
		}
		if o.CompositionRevision == "" {
			o.CompositionRevision = str(u.Object, append(p, "compositionRevisionRef", "name")...)
		}
		for _, raw := range slice(u.Object, append(p, "resourceRefs")...) {
			if m, ok := raw.(map[string]any); ok {
				if r := toRef(m); r.Name != "" {
					o.resourceRefs = append(o.resourceRefs, r)
				}
			}
		}
	}
	return o
}

// ProviderOf infers the cloud provider from an MR API group (e.g. dbforpostgresql.azure.upbound.io → azure).
func ProviderOf(apiVersion string) string {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return ""
	}
	parts := strings.Split(gv.Group, ".")
	for _, p := range parts {
		switch p {
		case "azure", "aws", "huawei", "gcp":
			return p
		}
	}
	return ""
}
