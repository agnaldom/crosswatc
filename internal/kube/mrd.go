package kube

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// Crossplane v2 ManagedResourceDefinitions.
	mrdGVR = schema.GroupVersionResource{Group: "apiextensions.crossplane.io", Version: "v1alpha1", Resource: "managedresourcedefinitions"}
	// Fallback for Crossplane v1: CRDs in category "managed".
	crdGVR = schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}
)

// MRType is a discovered Managed Resource type (spec §23).
type MRType struct {
	ResourceTarget
	Active bool // MRD spec.state == Active (always true for CRD fallback)
}

// MRIndex resolves Managed Resource types by GroupKind.
type MRIndex map[schema.GroupKind]MRType

// Lookup resolves an apiVersion/kind reference to its type.
func (ix MRIndex) Lookup(apiVersion, kind string) (MRType, bool) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return MRType{}, false
	}
	t, ok := ix[schema.GroupKind{Group: gv.Group, Kind: kind}]
	if ok && gv.Version != "" {
		t.Version = gv.Version // honour the version the reference uses
	}
	return t, ok
}

// Types returns the indexed types sorted by group/kind.
func (ix MRIndex) Types() []MRType {
	out := make([]MRType, 0, len(ix))
	for _, t := range ix {
		out = append(out, t)
	}
	slices.SortFunc(out, func(a, b MRType) int {
		return strings.Compare(a.Group+"/"+a.Kind, b.Group+"/"+b.Kind)
	})
	return out
}

// DiscoverMRDs indexes Managed Resource types from MRDs, falling back to CRDs in category "managed"
// when the MRD API is unavailable or empty. No provider is hardcoded.
func (c *Client) DiscoverMRDs(ctx context.Context) (MRIndex, error) {
	ix := MRIndex{}
	mrds, mrdErr := c.List(ctx, mrdGVR, ListOptions{})
	for i := range mrds {
		if t, ok := parseCRDLike(&mrds[i], false); ok {
			t.Active = str(mrds[i].Object, "spec", "state") != "Inactive"
			ix[schema.GroupKind{Group: t.Group, Kind: t.Kind}] = t
		}
	}
	if len(ix) > 0 {
		return ix, nil
	}
	crds, err := c.List(ctx, crdGVR, ListOptions{})
	if err != nil {
		if mrdErr != nil {
			return nil, fmt.Errorf("kube: discover MRDs: %w; CRD fallback: %w", mrdErr, err)
		}
		return nil, fmt.Errorf("kube: discover MRDs (CRD fallback): %w", err)
	}
	for i := range crds {
		if t, ok := parseCRDLike(&crds[i], true); ok {
			t.Active = true
			ix[schema.GroupKind{Group: t.Group, Kind: t.Kind}] = t
		}
	}
	return ix, nil
}

// parseCRDLike reads group/names/scope/versions shared by CRDs and MRDs.
// When requireManagedCategory is set, objects outside category "managed" are skipped.
func parseCRDLike(u *unstructured.Unstructured, requireManagedCategory bool) (MRType, bool) {
	spec := obj(u.Object, "spec")
	if requireManagedCategory {
		cats, _, _ := unstructured.NestedStringSlice(spec, "names", "categories")
		if !slices.Contains(cats, "managed") {
			return MRType{}, false
		}
	}
	t := MRType{ResourceTarget: ResourceTarget{
		Group:      str(spec, "group"),
		Kind:       str(spec, "names", "kind"),
		Plural:     str(spec, "names", "plural"),
		Namespaced: str(spec, "scope") == "Namespaced",
	}}
	var served string
	for _, raw := range slice(spec, "versions") {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if boolean(m, "storage") {
			t.Version = str(m, "name")
		}
		if served == "" && boolean(m, "served") {
			served = str(m, "name")
		}
	}
	if t.Version == "" {
		t.Version = served
	}
	if t.Group == "" || t.Kind == "" || t.Plural == "" || t.Version == "" {
		return MRType{}, false
	}
	return t, true
}

// ResolveMRType resolves an apiVersion/kind reference into an MRType.
// If not present in the MRIndex, it queries cluster CRDs dynamically to discover
// the resource plural and scope (even if missing category "managed"), and caches the result.
func (c *Client) ResolveMRType(ctx context.Context, mrs MRIndex, apiVersion, kind string) (MRType, bool) {
	if mrs != nil {
		if t, ok := mrs.Lookup(apiVersion, kind); ok {
			return t, true
		}
	}
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return MRType{}, false
	}
	gk := schema.GroupKind{Group: gv.Group, Kind: kind}

	var found *MRType
	// Dynamic lookup in cluster CRDs without requiring "managed" category
	if c != nil && c.Dynamic != nil {
		func() {
			defer func() { _ = recover() }()
			crds, err := c.List(ctx, crdGVR, ListOptions{})
			if err == nil {
				for i := range crds {
					spec := obj(crds[i].Object, "spec")
					if str(spec, "group") == gv.Group && str(spec, "names", "kind") == kind {
						plural := str(spec, "names", "plural")
						if plural == "" {
							plural = strings.ToLower(kind) + "s"
						}
						found = &MRType{
							ResourceTarget: ResourceTarget{
								Group:      gv.Group,
								Version:    gv.Version,
								Kind:       kind,
								Plural:     plural,
								Namespaced: str(spec, "scope") == "Namespaced",
							},
							Active: true,
						}
						return
					}
				}
			}
		}()
	}
	if found != nil {
		if mrs != nil {
			mrs[gk] = *found
		}
		return *found, true
	}
	return MRType{}, false
}
