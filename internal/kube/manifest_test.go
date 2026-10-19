package kube

import (
	"context"
	"errors"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestSanitizedYAML(t *testing.T) {
	u := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "db/v1", "kind": "FlexibleServer",
		"metadata": map[string]any{"name": "srv", "managedFields": []any{map[string]any{"manager": "x"}},
			"annotations": map[string]any{"kubectl.kubernetes.io/last-applied-configuration": "{secret}", "keep": "me"}},
		"spec":   map[string]any{"forProvider": map[string]any{"administratorLoginPassword": "hunter2", "skuName": "B1"}},
		"status": map[string]any{"atProvider": map[string]any{"fqdn": "srv.example"}},
	}}
	b, err := SanitizedYAML(u)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, bad := range []string{"hunter2", "managedFields", "last-applied", "{secret}"} {
		if strings.Contains(s, bad) {
			t.Errorf("leaked %q:\n%s", bad, s)
		}
	}
	for _, want := range []string{"administratorLoginPassword: '[REDACTED]'", "skuName: B1", "fqdn: srv.example", "keep: me"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q:\n%s", want, s)
		}
	}
	if u.Object["spec"].(map[string]any)["forProvider"].(map[string]any)["administratorLoginPassword"] != "hunter2" {
		t.Fatal("input mutated")
	}
}

func TestManifest(t *testing.T) {
	gvk := schema.GroupVersionKind{Group: "db.example.io", Version: "v1", Kind: "FlexibleServer"}
	m := meta.NewDefaultRESTMapper(nil)
	m.Add(gvk, meta.RESTScopeRoot)
	obj := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "db.example.io/v1", "kind": "FlexibleServer",
		"metadata": map[string]any{"name": "srv"}, "spec": map[string]any{"skuName": "B1"}}}
	c := &Client{Dynamic: fakeDynamic(nil, obj), Mapper: m}
	b, err := c.Manifest(context.Background(), Ref{APIVersion: "db.example.io/v1", Kind: "FlexibleServer", Name: "srv"})
	if err != nil || !strings.Contains(string(b), "skuName: B1") {
		t.Fatalf("%s %v", b, err)
	}
	if _, err := c.Manifest(context.Background(), Ref{APIVersion: "v1", Kind: "Secret", Name: "s"}); !errors.Is(err, ErrForbiddenKind) {
		t.Fatalf("secrets must be refused: %v", err)
	}
	if _, err := c.Manifest(context.Background(), Ref{APIVersion: "x/v1", Kind: "Unknown", Name: "u"}); err == nil {
		t.Fatal("unknown kind must error")
	}
}
