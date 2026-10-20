package kube

import (
	"context"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func TestParseMRDAndMRAP(t *testing.T) {
	mrd := parseMRD(&unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": "flexibleservers.dbforpostgresql.azure.upbound.io"},
		"spec":     map[string]any{"group": "dbforpostgresql.azure.upbound.io", "names": map[string]any{"kind": "FlexibleServer"}, "scope": "Cluster", "state": "Active"},
		"status":   map[string]any{"conditions": []any{map[string]any{"type": "Established", "status": "True"}}},
	}})
	if mrd.Kind != "FlexibleServer" || mrd.State != "Active" || !mrd.Active || !mrd.Established || mrd.Group != "dbforpostgresql.azure.upbound.io" {
		t.Fatalf("%+v", mrd)
	}
	if d := parseMRD(&unstructured.Unstructured{Object: map[string]any{"metadata": map[string]any{"name": "x"}, "spec": map[string]any{"state": "Inactive"}}}); d.Active || d.Established {
		t.Fatalf("inactive: %+v", d)
	}
	p := parseMRAP(&unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": "azure-db"},
		"spec":     map[string]any{"activate": []any{"*.dbforpostgresql.azure.upbound.io", int64(3)}},
	}})
	if p.Name != "azure-db" || len(p.Activate) != 1 || p.Activate[0] != "*.dbforpostgresql.azure.upbound.io" {
		t.Fatalf("%+v", p)
	}
}

func TestMRDsListAndMissingAPI(t *testing.T) {
	u := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apiextensions.crossplane.io/v1alpha1", "kind": "ManagedResourceDefinition",
		"metadata": map[string]any{"name": "b"}, "spec": map[string]any{"state": "Active"},
	}}
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{mrdGVR: "ManagedResourceDefinitionList", mrapGVR: "ManagedResourceActivationPolicyList"}, u)
	c := &Client{Dynamic: dyn}
	ms, err := c.MRDs(context.Background())
	if err != nil || len(ms) != 1 || ms[0].State != "Active" {
		t.Fatalf("%v %+v", err, ms)
	}
	ps, err := c.MRAPs(context.Background())
	if err != nil || len(ps) != 0 {
		t.Fatalf("%v %+v", err, ps)
	}

	if !IsMissingAPI(apierrors.NewNotFound(schema.GroupResource{Group: "apiextensions.crossplane.io", Resource: "managedresourcedefinitions"}, "")) {
		t.Fatal("404 is a missing API")
	}
	if !IsMissingAPI(&apimeta.NoKindMatchError{GroupKind: schema.GroupKind{Kind: "X"}}) {
		t.Fatal("no match is a missing API")
	}
	if IsMissingAPI(apierrors.NewForbidden(schema.GroupResource{}, "", nil)) {
		t.Fatal("forbidden is a real error")
	}
}
