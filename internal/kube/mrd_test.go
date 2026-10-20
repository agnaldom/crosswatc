package kube

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func crdLike(apiVersion, kind, group, mrKind, plural, scope string, categories []any, extra map[string]any) *unstructured.Unstructured {
	spec := map[string]any{
		"group": group, "scope": scope,
		"names": map[string]any{"kind": mrKind, "plural": plural, "categories": categories},
		"versions": []any{
			map[string]any{"name": "v1beta1", "served": true, "storage": false},
			map[string]any{"name": "v1beta2", "served": true, "storage": true},
		},
	}
	for k, v := range extra {
		spec[k] = v
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": apiVersion, "kind": kind,
		"metadata": map[string]any{"name": plural + "." + group},
		"spec":     spec,
	}}
}

var mrdListKinds = map[schema.GroupVersionResource]string{
	mrdGVR: "ManagedResourceDefinitionList",
	crdGVR: "CustomResourceDefinitionList",
}

func TestDiscoverMRDsFromMRDs(t *testing.T) {
	c := &Client{Dynamic: fakeDynamic(mrdListKinds,
		crdLike("apiextensions.crossplane.io/v1alpha1", "ManagedResourceDefinition", "dbforpostgresql.azure.upbound.io",
			"FlexibleServer", "flexibleservers", "Cluster", nil, map[string]any{"state": "Active"}),
		crdLike("apiextensions.crossplane.io/v1alpha1", "ManagedResourceDefinition", "s3.aws.upbound.io",
			"Bucket", "buckets", "Namespaced", nil, map[string]any{"state": "Inactive"}),
		// CRD must be ignored when MRDs exist
		crdLike("apiextensions.k8s.io/v1", "CustomResourceDefinition", "network.azure.upbound.io",
			"VirtualNetwork", "virtualnetworks", "Cluster", []any{"crossplane", "managed"}, nil),
	)}
	ix, err := c.DiscoverMRDs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(ix) != 2 {
		t.Fatalf("got %d types: %v", len(ix), ix.Types())
	}
	fs, ok := ix.Lookup("dbforpostgresql.azure.upbound.io/v1beta1", "FlexibleServer")
	if !ok || fs.Plural != "flexibleservers" || fs.Version != "v1beta1" || !fs.Active || fs.Namespaced {
		t.Fatalf("flexibleserver: %+v", fs)
	}
	b, _ := ix.Lookup("s3.aws.upbound.io/v1beta2", "Bucket")
	if b.Active || !b.Namespaced {
		t.Fatalf("bucket: %+v", b)
	}
	if ts := ix.Types(); ts[0].Group != "dbforpostgresql.azure.upbound.io" {
		t.Fatalf("not sorted: %v", ts)
	}
}

func TestDiscoverMRDsFallbackToCRDs(t *testing.T) {
	c := &Client{Dynamic: fakeDynamic(mrdListKinds,
		crdLike("apiextensions.k8s.io/v1", "CustomResourceDefinition", "network.azure.upbound.io",
			"VirtualNetwork", "virtualnetworks", "Cluster", []any{"crossplane", "managed", "azure"}, nil),
		crdLike("apiextensions.k8s.io/v1", "CustomResourceDefinition", "cert-manager.io",
			"Certificate", "certificates", "Namespaced", []any{"cert-manager"}, nil),
	)}
	ix, err := c.DiscoverMRDs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(ix) != 1 {
		t.Fatalf("only category=managed must be indexed: %v", ix.Types())
	}
	vn, ok := ix.Lookup("network.azure.upbound.io/v1beta2", "VirtualNetwork")
	if !ok || vn.Version != "v1beta2" || !vn.Active {
		t.Fatalf("vnet: %+v", vn)
	}
	if _, ok := ix.Lookup("::bad", "X"); ok {
		t.Fatal("bad apiVersion must not resolve")
	}
}

func TestParseCRDLikeRejectsIncomplete(t *testing.T) {
	u := crdLike("apiextensions.k8s.io/v1", "CustomResourceDefinition", "", "X", "xs", "Cluster", []any{"managed"}, nil)
	if _, ok := parseCRDLike(u, true); ok {
		t.Fatal("missing group must be rejected")
	}
}

func TestResolveMRTypeDynamic(t *testing.T) {
	c := &Client{Dynamic: fakeDynamic(mrdListKinds,
		crdLike("apiextensions.k8s.io/v1", "CustomResourceDefinition", "cert-manager.io",
			"Certificate", "certificates", "Namespaced", []any{"cert-manager"}, nil),
	)}

	ix := MRIndex{}

	// 1. Dynamic lookup against CRD without category=managed
	certType, ok := c.ResolveMRType(context.Background(), ix, "cert-manager.io/v1", "Certificate")
	if !ok {
		t.Fatal("expected Certificate to resolve dynamically via CRD")
	}
	if certType.Plural != "certificates" || !certType.Namespaced || certType.Group != "cert-manager.io" {
		t.Fatalf("unexpected resolved certType: %+v", certType)
	}

	// 2. Subsequent lookup hits cache in ix
	cached, ok := ix.Lookup("cert-manager.io/v1", "Certificate")
	if !ok || cached.Plural != "certificates" {
		t.Fatalf("expected cached type in ix: %+v", cached)
	}

	// 3. When CRD does not exist, returns false
	_, ok = c.ResolveMRType(context.Background(), ix, "unknown.io/v1", "UnknownThing")
	if ok {
		t.Fatal("expected UnknownThing to fail resolving")
	}
}
