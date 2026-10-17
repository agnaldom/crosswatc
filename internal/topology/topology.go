// Package topology builds the Claim → XR → Managed Resources graph (spec §28, §29).
package topology

import (
	"strings"

	"github.com/agnaldom/crosswatc/internal/kube"
)

// Node types.
const (
	TypeClaim     = "claim"
	TypeComposite = "composite"
	TypeManaged   = "managed-resource"
)

// Node statuses.
const (
	StatusReady    = "ready"
	StatusNotReady = "not-ready"
	StatusError    = "error"
	StatusMissing  = "missing"
	StatusDeleting = "deleting"
)

// Node is a vertex of the topology graph.
type Node struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	Namespace    string `json:"namespace,omitempty"`
	APIVersion   string `json:"apiVersion"`
	Status       string `json:"status"`
	Provider     string `json:"provider,omitempty"`
	ExternalName string `json:"externalName,omitempty"`
	Role         string `json:"compositionResourceName,omitempty"`
	Message      string `json:"message,omitempty"`
}

// Edge links a composing object to a composed one.
type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// Graph is the §29 response body.
type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// Build derives the graph from a discovered resource. A Crossplane v2 namespaced XR
// (claim and XR are the same object) yields a single composite root.
func Build(d kube.DiscoveredResource) Graph {
	g := Graph{Nodes: []Node{}, Edges: []Edge{}}
	seen := map[string]bool{}
	add := func(n Node, parent string) {
		if seen[n.ID] {
			return
		}
		seen[n.ID] = true
		g.Nodes = append(g.Nodes, n)
		if parent != "" && parent != n.ID {
			g.Edges = append(g.Edges, Edge{Source: parent, Target: n.ID})
		}
	}

	sameObject := d.XR != nil && d.XR.ID() == d.Claim.ID()
	if !sameObject {
		add(node(d.Claim, TypeClaim), "")
	}
	if d.XR != nil {
		parent := d.XR.Parent
		if sameObject {
			parent = ""
		}
		add(node(*d.XR, TypeComposite), parent)
	}
	for _, c := range d.Composites {
		add(node(c, TypeComposite), c.Parent)
	}
	for _, m := range d.MRs {
		n := node(m, TypeManaged)
		n.Provider = kube.ProviderOf(m.APIVersion)
		add(n, m.Parent)
	}
	for _, m := range d.Missing {
		add(Node{
			ID: m.ID(), Type: typeOfMissing(d, m), Kind: m.Kind, Name: m.Name, Namespace: m.Namespace,
			APIVersion: m.APIVersion, Status: StatusMissing, Provider: kube.ProviderOf(m.APIVersion),
			Message: "referenced object not found",
		}, m.Parent)
	}
	return g
}

func typeOfMissing(d kube.DiscoveredResource, m kube.MissingRef) string {
	if m.Parent == d.Claim.ID() && d.XR == nil {
		return TypeComposite
	}
	return TypeManaged
}

func node(o kube.Object, typ string) Node {
	n := Node{
		ID: o.ID(), Type: typ, Kind: o.Kind, Name: o.Name, Namespace: o.Namespace, APIVersion: o.APIVersion,
		ExternalName: o.ExternalName, Role: o.CompositionResourceName,
	}
	n.Status, n.Message = status(o)
	return n
}

func status(o kube.Object) (string, string) {
	if o.Deleting {
		return StatusDeleting, ""
	}
	for _, t := range []string{"Synced", "Ready"} {
		if c, ok := o.Conditions.Get(t); ok && c.Status == "False" && strings.Contains(c.Reason, "Error") {
			return StatusError, c.Message
		}
	}
	if o.Ready() {
		return StatusReady, ""
	}
	if c, ok := o.Conditions.Get("Ready"); ok {
		return StatusNotReady, c.Reason
	}
	return StatusNotReady, ""
}
