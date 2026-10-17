package topology

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/agnaldom/crosswatc/internal/domain"
	"github.com/agnaldom/crosswatc/internal/kube"
)

func obj(apiVersion, kind, ns, name, parent string, conds ...domain.Condition) kube.Object {
	return kube.Object{Ref: kube.Ref{APIVersion: apiVersion, Kind: kind, Namespace: ns, Name: name}, Parent: parent, Conditions: conds}
}

var (
	ready    = domain.Condition{Type: "Ready", Status: "True"}
	creating = domain.Condition{Type: "Ready", Status: "False", Reason: "Creating"}
	errored  = domain.Condition{Type: "Synced", Status: "False", Reason: "ReconcileError", Message: "NicInUse"}
)

func TestBuildClaimXRMR(t *testing.T) {
	claim := obj("database.crosswatc.io/v1", "PostgreSQLInstance", "team", "pgsql-payments", "", ready)
	xr := obj("database.crosswatc.io/v1", "XPostgreSQLInstance", "", "xpostgresql-92a", claim.ID(), ready)
	net := obj("net.crosswatc.io/v1", "XNetwork", "", "xnet", xr.ID(), creating)
	server := obj("dbforpostgresql.azure.upbound.io/v1beta1", "FlexibleServer", "", "postgresql-server", xr.ID(), ready)
	server.ExternalName = "pg-payments"
	pe := obj("network.azure.upbound.io/v1beta1", "PrivateEndpoint", "", "pe", net.ID(), errored)
	del := obj("network.azure.upbound.io/v1beta1", "Subnet", "", "sn", net.ID(), ready)
	del.Deleting = true

	g := Build(kube.DiscoveredResource{
		Claim: claim, XR: &xr, Composites: []kube.Object{net}, MRs: []kube.Object{server, pe, del},
		Missing: []kube.MissingRef{{Ref: kube.Ref{APIVersion: "network.azure.upbound.io/v1beta1", Kind: "VirtualNetwork", Name: "vnet"}, Parent: xr.ID()}},
	})

	if len(g.Nodes) != 7 || len(g.Edges) != 6 {
		t.Fatalf("nodes=%d edges=%d", len(g.Nodes), len(g.Edges))
	}
	by := map[string]Node{}
	for _, n := range g.Nodes {
		by[n.Name] = n
	}
	checks := []struct{ name, typ, status, provider string }{
		{"pgsql-payments", TypeClaim, StatusReady, ""},
		{"xpostgresql-92a", TypeComposite, StatusReady, ""},
		{"xnet", TypeComposite, StatusNotReady, ""},
		{"postgresql-server", TypeManaged, StatusReady, "azure"},
		{"pe", TypeManaged, StatusError, "azure"},
		{"sn", TypeManaged, StatusDeleting, "azure"},
		{"vnet", TypeManaged, StatusMissing, "azure"},
	}
	for _, c := range checks {
		n := by[c.name]
		if n.Type != c.typ || n.Status != c.status || n.Provider != c.provider {
			t.Errorf("%s: %+v", c.name, n)
		}
	}
	if by["pe"].Message != "NicInUse" || by["xnet"].Message != "Creating" || by["postgresql-server"].ExternalName != "pg-payments" {
		t.Fatal("messages/external name not propagated")
	}
	want := map[Edge]bool{
		{claim.ID(), xr.ID()}: true, {xr.ID(), net.ID()}: true, {xr.ID(), server.ID()}: true,
		{net.ID(), pe.ID()}: true, {net.ID(), del.ID()}: true, {xr.ID(), "VirtualNetwork/vnet"}: true,
	}
	for _, e := range g.Edges {
		if !want[e] {
			t.Errorf("unexpected edge %+v", e)
		}
	}

	b, _ := json.Marshal(g)
	if !strings.Contains(string(b), `"type":"managed-resource"`) || !strings.Contains(string(b), `"source":"PostgreSQLInstance/team/pgsql-payments"`) {
		t.Fatalf("json shape: %s", b)
	}
}

func TestBuildNamespacedXRSingleRoot(t *testing.T) {
	root := obj("storage.crosswatc.io/v1", "Bucket", "team", "logs", "", ready)
	xr := root
	xr.Role = kube.RoleComposite
	mr := obj("storage.azure.m.upbound.io/v1beta1", "Account", "team", "acct", root.ID(), ready)
	g := Build(kube.DiscoveredResource{Claim: root, XR: &xr, MRs: []kube.Object{mr}})
	if len(g.Nodes) != 2 || len(g.Edges) != 1 || g.Nodes[0].Type != TypeComposite {
		t.Fatalf("%+v", g)
	}
}

func TestBuildPendingXR(t *testing.T) {
	claim := obj("database.crosswatc.io/v1", "PostgreSQLInstance", "team", "pg", "")
	g := Build(kube.DiscoveredResource{Claim: claim, Missing: []kube.MissingRef{{Ref: kube.Ref{Kind: "XPostgreSQLInstance", Name: "x"}, Parent: claim.ID()}}})
	if len(g.Nodes) != 2 || g.Nodes[1].Type != TypeComposite || g.Nodes[1].Status != StatusMissing || g.Nodes[0].Status != StatusNotReady {
		t.Fatalf("%+v", g)
	}
	empty := Build(kube.DiscoveredResource{Claim: claim})
	b, _ := json.Marshal(empty)
	if !strings.Contains(string(b), `"edges":[]`) {
		t.Fatalf("edges must serialize as []: %s", b)
	}
}
