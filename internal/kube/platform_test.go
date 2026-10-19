package kube

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestPlatformCatalog(t *testing.T) {
	prov := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "pkg.crossplane.io/v1", "kind": "Provider", "metadata": map[string]any{"name": "provider-azure-network"},
		"spec":   map[string]any{"package": "xpkg.upbound.io/upbound/provider-azure-network:v1.11.0", "revisionActivationPolicy": "Automatic"},
		"status": map[string]any{"currentRevision": "provider-azure-network-abc", "conditions": []any{map[string]any{"type": "Healthy", "status": "False", "reason": "UnhealthyPackageRevision"}}},
	}}
	fnOld := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "pkg.crossplane.io/v1beta1", "kind": "Function", "metadata": map[string]any{"name": "function-patch-and-transform"},
		"spec": map[string]any{"package": "xpkg.crossplane.io/crossplane-contrib/function-patch-and-transform:v0.8.2"},
	}}
	pipe := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apiextensions.crossplane.io/v1", "kind": "Composition", "metadata": map[string]any{"name": "pg-azure"},
		"spec": map[string]any{"compositeTypeRef": map[string]any{"apiVersion": "db.crosswatc.io/v1", "kind": "XPostgreSQLInstance"},
			"mode": "Pipeline", "pipeline": []any{
				map[string]any{"step": "render", "functionRef": map[string]any{"name": "function-go-templating"}},
				map[string]any{"step": "patch", "functionRef": map[string]any{"name": "function-patch-and-transform"}},
				map[string]any{"step": "ready", "functionRef": map[string]any{"name": "function-go-templating"}},
			}},
	}}
	legacy := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apiextensions.crossplane.io/v1", "kind": "Composition", "metadata": map[string]any{"name": "a-legacy"},
		"spec": map[string]any{"compositeTypeRef": map[string]any{"apiVersion": "x/v1", "kind": "X"}, "resources": []any{map[string]any{}, map[string]any{}}},
	}}
	c := &Client{Dynamic: fakeDynamic(map[schema.GroupVersionResource]string{
		packageGVRs["Provider"][0]: "ProviderList", packageGVRs["Function"][0]: "FunctionList", packageGVRs["Function"][1]: "FunctionList",
		packageGVRs["Configuration"][0]: "ConfigurationList", compositionGVR: "CompositionList",
	}, prov, fnOld, pipe, legacy)}
	ctx := context.Background()

	ps, err := c.Packages(ctx, "Provider")
	if err != nil || len(ps) != 1 || ps[0].CurrentRevision != "provider-azure-network-abc" || ps[0].Conditions[0].Reason != "UnhealthyPackageRevision" || ps[0].RevisionPolicy != "Automatic" {
		t.Fatalf("%+v %v", ps, err)
	}
	// v1 answers, so no fallback to v1beta1 (an API server serves the same objects under both versions)
	if fs, err := c.Packages(ctx, "Function"); err != nil || fs == nil {
		t.Fatalf("%+v %v", fs, err)
	}
	if _, err := c.Packages(ctx, "Nope"); err == nil {
		t.Fatal("unknown kind")
	}
	cs, err := c.Compositions(ctx)
	if err != nil || len(cs) != 2 {
		t.Fatal(cs, err)
	}
	if l := cs[0]; l.Name != "a-legacy" || l.Mode != "Resources" || l.Steps != 2 {
		t.Fatalf("%+v", l)
	}
	if p := cs[1]; p.Mode != "Pipeline" || p.Steps != 3 || len(p.Functions) != 2 || p.CompositeType != "db.crosswatc.io/v1/XPostgreSQLInstance" {
		t.Fatalf("%+v", p)
	}
}
