package kube

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynfake "k8s.io/client-go/dynamic/fake"
)

func xrdObject(version, name string, spec map[string]any) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apiextensions.crossplane.io/" + version,
		"kind":       "CompositeResourceDefinition",
		"metadata":   map[string]any{"name": name},
		"spec":       spec,
		"status":     map[string]any{"conditions": []any{map[string]any{"type": "Established", "status": "True"}}},
	}}
}

func postgresXRDSpec() map[string]any {
	return map[string]any{
		"group":      "database.crosswatc.io",
		"names":      map[string]any{"kind": "XPostgreSQLInstance", "plural": "xpostgresqlinstances"},
		"claimNames": map[string]any{"kind": "PostgreSQLInstance", "plural": "postgresqlinstances"},
		"versions": []any{
			map[string]any{"name": "v1alpha1", "served": true, "referenceable": false},
			map[string]any{"name": "v1beta1", "served": true, "referenceable": true, "schema": map[string]any{
				"openAPIV3Schema": map[string]any{"type": "object", "properties": map[string]any{
					"spec": map[string]any{"type": "object", "properties": map[string]any{
						"parameters": map[string]any{"type": "object", "properties": map[string]any{
							"vmSize": map[string]any{"type": "string"},
						}},
					}},
				}},
			}},
		},
	}
}

func fakeDynamic(gvrs map[schema.GroupVersionResource]string, objs ...runtime.Object) *dynfake.FakeDynamicClient {
	return dynfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrs, objs...)
}

var xrdListKinds = map[schema.GroupVersionResource]string{
	xrdGVRs[0]: "CompositeResourceDefinitionList",
	xrdGVRs[1]: "CompositeResourceDefinitionList",
}

func TestDiscoverXRDsV1(t *testing.T) {
	ns := map[string]any{
		"group": "app.crosswatc.io", "scope": "Namespaced",
		"names":    map[string]any{"kind": "Bucket", "plural": "buckets"},
		"versions": []any{map[string]any{"name": "v1", "served": true, "referenceable": true}},
	}
	c := &Client{Dynamic: fakeDynamic(xrdListKinds,
		xrdObject("v1", "xpostgresqlinstances.database.crosswatc.io", postgresXRDSpec()),
		xrdObject("v1", "buckets.app.crosswatc.io", ns))}
	xrds, err := c.DiscoverXRDs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// the fake serves both versions from the same tracker; v2 is tried first
	if len(xrds) != 2 {
		t.Fatalf("got %d XRDs", len(xrds))
	}
	byKind := map[string]XRD{}
	for _, x := range xrds {
		byKind[x.Kind] = x
	}

	pg := byKind["XPostgreSQLInstance"]
	if pg.Version() != "v1beta1" || pg.Scope != "Cluster" || !pg.Established {
		t.Fatalf("pg: %+v", pg)
	}
	ct, ok := pg.ClaimTarget()
	if !ok || ct.GVR() != (schema.GroupVersionResource{Group: "database.crosswatc.io", Version: "v1beta1", Resource: "postgresqlinstances"}) || !ct.Namespaced {
		t.Fatalf("claim target %+v", ct)
	}
	if pg.XRTarget().Namespaced {
		t.Fatal("cluster XR must not be namespaced")
	}
	props, _, _ := unstructured.NestedMap(pg.Schema(""), "properties", "spec", "properties", "parameters", "properties")
	if _, ok := props["vmSize"]; !ok {
		t.Fatalf("schema not extracted: %v", pg.Schema(""))
	}
	if pg.Schema("v1alpha1") != nil || pg.Schema("nope") != nil {
		t.Fatal("unexpected schema for version without schema")
	}

	b := byKind["Bucket"]
	if _, ok := b.ClaimTarget(); ok {
		t.Fatal("namespaced XR has no claim")
	}
	uf, ok := b.UserFacingTarget()
	if !ok || !uf.Namespaced || uf.Kind != "Bucket" {
		t.Fatalf("user-facing %+v", uf)
	}
}

func TestXRDVersionFallbacks(t *testing.T) {
	x := XRD{Versions: []XRDVersion{{Name: "a"}, {Name: "b", Served: true}}}
	if x.Version() != "b" {
		t.Fatal("expected first served")
	}
	if (XRD{}).Version() != "" {
		t.Fatal("expected empty")
	}
	if _, ok := (XRD{Scope: "Cluster"}).UserFacingTarget(); ok {
		t.Fatal("cluster XR without claim has no user-facing target")
	}
}
