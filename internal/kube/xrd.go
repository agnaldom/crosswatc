package kube

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// XRD API versions tried in order (Crossplane v2 serves v2; v1 remains for v1.x).
var xrdGVRs = []schema.GroupVersionResource{
	{Group: "apiextensions.crossplane.io", Version: "v2", Resource: "compositeresourcedefinitions"},
	{Group: "apiextensions.crossplane.io", Version: "v1", Resource: "compositeresourcedefinitions"},
}

// XRD is a discovered CompositeResourceDefinition (spec §22).
type XRD struct {
	Name        string
	Group       string
	Kind        string
	Plural      string
	ClaimKind   string // empty when the XRD offers no claim (e.g. Crossplane v2 namespaced XRs)
	ClaimPlural string
	Scope       string // Cluster | Namespaced | LegacyCluster
	Established bool
	Versions    []XRDVersion
}

// XRDVersion is one version of an XRD with its OpenAPI schema.
type XRDVersion struct {
	Name          string
	Served        bool
	Referenceable bool
	Schema        map[string]any // openAPIV3Schema
}

// Version returns the referenceable version, else the first served one.
func (x XRD) Version() string {
	for _, v := range x.Versions {
		if v.Referenceable {
			return v.Name
		}
	}
	for _, v := range x.Versions {
		if v.Served {
			return v.Name
		}
	}
	return ""
}

// Schema returns the OpenAPI schema of version v (or of Version() when v is empty).
func (x XRD) Schema(v string) map[string]any {
	if v == "" {
		v = x.Version()
	}
	for _, ver := range x.Versions {
		if ver.Name == v {
			return ver.Schema
		}
	}
	return nil
}

// XRTarget is the composite resource type.
func (x XRD) XRTarget() ResourceTarget {
	return ResourceTarget{Group: x.Group, Version: x.Version(), Kind: x.Kind, Plural: x.Plural, Namespaced: x.Scope == "Namespaced"}
}

// ClaimTarget is the claim type, if the XRD offers claims.
func (x XRD) ClaimTarget() (ResourceTarget, bool) {
	if x.ClaimKind == "" {
		return ResourceTarget{}, false
	}
	return ResourceTarget{Group: x.Group, Version: x.Version(), Kind: x.ClaimKind, Plural: x.ClaimPlural, Namespaced: true}, true
}

// UserFacingTarget is the namespaced object users own: the claim, or a namespaced XR (Crossplane v2).
func (x XRD) UserFacingTarget() (ResourceTarget, bool) {
	if t, ok := x.ClaimTarget(); ok {
		return t, true
	}
	if x.Scope == "Namespaced" {
		return x.XRTarget(), true
	}
	return ResourceTarget{}, false
}

// DiscoverXRDs lists every XRD in the cluster.
func (c *Client) DiscoverXRDs(ctx context.Context) ([]XRD, error) {
	var lastErr error
	for i, gvr := range xrdGVRs {
		items, err := c.List(ctx, gvr, ListOptions{})
		if err != nil {
			lastErr = err
			continue
		}
		if len(items) == 0 && i < len(xrdGVRs)-1 {
			continue // version served but empty: try the next one
		}
		out := make([]XRD, 0, len(items))
		for i := range items {
			out = append(out, ParseXRD(&items[i]))
		}
		return out, nil
	}
	if lastErr == nil {
		return nil, nil
	}
	return nil, fmt.Errorf("kube: discover XRDs: %w", lastErr)
}

// ParseXRD extracts the fields of spec §22.
func ParseXRD(u *unstructured.Unstructured) XRD {
	spec := obj(u.Object, "spec")
	x := XRD{
		Name:        u.GetName(),
		Group:       str(spec, "group"),
		Kind:        str(spec, "names", "kind"),
		Plural:      str(spec, "names", "plural"),
		ClaimKind:   str(spec, "claimNames", "kind"),
		ClaimPlural: str(spec, "claimNames", "plural"),
		Scope:       str(spec, "scope"),
		Established: Conditions(u).IsTrue("Established"),
	}
	if x.Scope == "" {
		x.Scope = "Cluster"
	}
	for _, raw := range slice(spec, "versions") {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		x.Versions = append(x.Versions, XRDVersion{
			Name:          str(m, "name"),
			Served:        boolean(m, "served"),
			Referenceable: boolean(m, "referenceable"),
			Schema:        obj(m, "schema", "openAPIV3Schema"),
		})
	}
	return x
}
