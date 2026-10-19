package kube

import (
	"context"
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/agnaldom/crosswatc/internal/domain"
)

// Crossplane package kinds, tried newest version first.
var packageGVRs = map[string][]schema.GroupVersionResource{
	"Provider":      {{Group: "pkg.crossplane.io", Version: "v1", Resource: "providers"}},
	"Function":      {{Group: "pkg.crossplane.io", Version: "v1", Resource: "functions"}, {Group: "pkg.crossplane.io", Version: "v1beta1", Resource: "functions"}},
	"Configuration": {{Group: "pkg.crossplane.io", Version: "v1", Resource: "configurations"}},
}

var compositionGVR = schema.GroupVersionResource{Group: "apiextensions.crossplane.io", Version: "v1", Resource: "compositions"}

// Package is an installed Crossplane package (Provider, Function or Configuration).
type Package struct {
	Kind              string            `json:"kind"`
	Name              string            `json:"name"`
	Package           string            `json:"package"`
	CurrentRevision   string            `json:"currentRevision,omitempty"`
	CurrentIdentifier string            `json:"currentIdentifier,omitempty"`
	RevisionPolicy    string            `json:"revisionActivationPolicy,omitempty"`
	Conditions        domain.Conditions `json:"conditions"`
}

// Composition summarizes a Composition: target XR type, mode and pipeline functions.
type Composition struct {
	Name          string   `json:"name"`
	CompositeType string   `json:"compositeType"` // apiVersion/kind
	Mode          string   `json:"mode"`          // Pipeline | Resources
	Functions     []string `json:"functions,omitempty"`
	Steps         int      `json:"steps"`
	Revision      string   `json:"latestRevision,omitempty"`
}

// Packages lists installed packages of kind (Provider, Function, Configuration).
func (c *Client) Packages(ctx context.Context, kind string) ([]Package, error) {
	gvrs, ok := packageGVRs[kind]
	if !ok {
		return nil, fmt.Errorf("kube: unknown package kind %q", kind)
	}
	var lastErr error
	for _, gvr := range gvrs {
		items, err := c.List(ctx, gvr, ListOptions{})
		if err != nil {
			lastErr = err
			continue
		}
		out := make([]Package, 0, len(items))
		for i := range items {
			out = append(out, parsePackage(&items[i], kind))
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
		return out, nil
	}
	return nil, lastErr
}

func parsePackage(u *unstructured.Unstructured, kind string) Package {
	return Package{
		Kind: kind, Name: u.GetName(), Package: str(u.Object, "spec", "package"),
		CurrentRevision:   str(u.Object, "status", "currentRevision"),
		CurrentIdentifier: str(u.Object, "status", "currentIdentifier"),
		RevisionPolicy:    str(u.Object, "spec", "revisionActivationPolicy"),
		Conditions:        Conditions(u),
	}
}

// Compositions lists Compositions.
func (c *Client) Compositions(ctx context.Context) ([]Composition, error) {
	items, err := c.List(ctx, compositionGVR, ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]Composition, 0, len(items))
	for i := range items {
		out = append(out, parseComposition(&items[i]))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func parseComposition(u *unstructured.Unstructured) Composition {
	spec := obj(u.Object, "spec")
	c := Composition{
		Name:          u.GetName(),
		CompositeType: str(spec, "compositeTypeRef", "apiVersion") + "/" + str(spec, "compositeTypeRef", "kind"),
		Mode:          str(spec, "mode"),
		Revision:      str(u.Object, "status", "latestRevision"),
	}
	if c.Mode == "" {
		c.Mode = "Resources"
		if len(slice(spec, "pipeline")) > 0 {
			c.Mode = "Pipeline"
		}
	}
	seen := map[string]bool{}
	for _, raw := range slice(spec, "pipeline") {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		c.Steps++
		if f := str(m, "functionRef", "name"); f != "" && !seen[f] {
			seen[f] = true
			c.Functions = append(c.Functions, f)
		}
	}
	if c.Mode == "Resources" {
		c.Steps = len(slice(spec, "resources"))
	}
	return c
}
