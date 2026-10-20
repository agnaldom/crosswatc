package kube

import (
	"context"
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/agnaldom/crosswatc/internal/domain"
)

// Crossplane v2 activation policies (MRDs share mrdGVR with discovery in mrd.go).
var mrapGVR = schema.GroupVersionResource{Group: "apiextensions.crossplane.io", Version: "v1alpha1", Resource: "managedresourceactivationpolicies"}

// MRD is a ManagedResourceDefinition: a provider's managed resource kind and whether it is active.
type MRD struct {
	Name        string            `json:"name"`
	Group       string            `json:"group"`
	Kind        string            `json:"kind"`
	Scope       string            `json:"scope,omitempty"`
	State       string            `json:"state,omitempty"` // spec.state as stored: Active | Inactive
	Active      bool              `json:"active"`          // same rule as DiscoverMRDs: anything but Inactive
	Established bool              `json:"established"`
	Conditions  domain.Conditions `json:"conditions"`
}

// MRAP is a ManagedResourceActivationPolicy: which MRDs it activates.
type MRAP struct {
	Name       string            `json:"name"`
	Activate   []string          `json:"activate"`
	Conditions domain.Conditions `json:"conditions"`
}

// MRDs lists ManagedResourceDefinitions. Returns an IsMissingAPI error on Crossplane v1.
func (c *Client) MRDs(ctx context.Context) ([]MRD, error) {
	items, err := c.List(ctx, mrdGVR, ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]MRD, 0, len(items))
	for i := range items {
		out = append(out, parseMRD(&items[i]))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func parseMRD(u *unstructured.Unstructured) MRD {
	m := MRD{
		Name: u.GetName(), Group: str(u.Object, "spec", "group"), Kind: str(u.Object, "spec", "names", "kind"),
		Scope: str(u.Object, "spec", "scope"), State: str(u.Object, "spec", "state"), Conditions: Conditions(u),
	}
	m.Active = m.State != "Inactive"
	for _, cd := range m.Conditions {
		if cd.Type == "Established" && cd.Status == "True" {
			m.Established = true
		}
	}
	return m
}

// MRAPs lists ManagedResourceActivationPolicies. Returns an IsMissingAPI error on Crossplane v1.
func (c *Client) MRAPs(ctx context.Context) ([]MRAP, error) {
	items, err := c.List(ctx, mrapGVR, ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]MRAP, 0, len(items))
	for i := range items {
		out = append(out, parseMRAP(&items[i]))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func parseMRAP(u *unstructured.Unstructured) MRAP {
	p := MRAP{Name: u.GetName(), Activate: []string{}, Conditions: Conditions(u)}
	for _, a := range slice(u.Object, "spec", "activate") {
		if s, ok := a.(string); ok {
			p.Activate = append(p.Activate, s)
		}
	}
	return p
}
