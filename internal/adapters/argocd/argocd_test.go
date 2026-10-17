package argocd

import (
	"context"
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/agnaldom/crosswatc/internal/contract"
	"github.com/agnaldom/crosswatc/internal/lifecycle"
)

var errNotFound = errors.New("not found")

type fakeGetter map[string]*unstructured.Unstructured

func (f fakeGetter) Get(_ context.Context, gvr schema.GroupVersionResource, ns, name string) (*unstructured.Unstructured, error) {
	if gvr != ApplicationGVR {
		return nil, errors.New("unexpected gvr")
	}
	if u, ok := f[ns+"/"+name]; ok {
		return u, nil
	}
	return nil, errNotFound
}

func app(status map[string]any) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{"apiVersion": "argoproj.io/v1alpha1", "kind": "Application", "status": status}}
}

func TestResolveApp(t *testing.T) {
	a := New(nil, "argocd", nil)
	cases := []struct {
		name   string
		labels map[string]string
		ann    map[string]string
		want   AppRef
		ok     bool
	}{
		{"explicit name", nil, map[string]string{contract.AnnotationArgoApp: "claims-dev"}, AppRef{"argocd", "claims-dev"}, true},
		{"explicit ns/name", nil, map[string]string{contract.AnnotationArgoApp: "team/claims"}, AppRef{"team", "claims"}, true},
		{"tracking id", nil, map[string]string{trackingIDAnnotation: "claims-dev:database.crosswatc.io/PostgreSQLInstance:team/pg"}, AppRef{"argocd", "claims-dev"}, true},
		{"tracking id any-namespace", nil, map[string]string{trackingIDAnnotation: "team-a_claims:g/K:ns/n"}, AppRef{"team-a", "claims"}, true},
		{"instance label", map[string]string{instanceLabel: "claims-prd"}, nil, AppRef{"argocd", "claims-prd"}, true},
		{"annotation wins", map[string]string{instanceLabel: "other"}, map[string]string{contract.AnnotationArgoApp: "claims-dev"}, AppRef{"argocd", "claims-dev"}, true},
		{"none", nil, nil, AppRef{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := a.ResolveApp(tc.labels, tc.ann)
			if got != tc.want || ok != tc.ok {
				t.Fatalf("got %v %v, want %v %v", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestStatus(t *testing.T) {
	g := fakeGetter{
		"argocd/claims-dev": app(map[string]any{
			"sync":           map[string]any{"status": "Synced", "revision": "abc"},
			"health":         map[string]any{"status": "Healthy"},
			"operationState": map[string]any{"phase": "Succeeded", "message": "successfully synced"},
		}),
		"argocd/multi": app(map[string]any{
			"sync":           map[string]any{"status": "OutOfSync", "revisions": []any{"r1", "r2"}},
			"operationState": map[string]any{"phase": "Failed", "message": "one or more objects failed"},
		}),
	}
	a := New(g, "argocd", func(err error) bool { return errors.Is(err, errNotFound) })

	s, err := a.Status(context.Background(), AppRef{"argocd", "claims-dev"})
	want := lifecycle.ArgoSignal{Found: true, SyncStatus: "Synced", Health: "Healthy", OperationPhase: "Succeeded", SyncedRevision: "abc", Message: "successfully synced"}
	if err != nil || s != want {
		t.Fatalf("got %+v %v", s, err)
	}
	s, _ = a.Status(context.Background(), AppRef{"argocd", "multi"})
	if s.SyncedRevision != "r1" || s.OperationPhase != "Failed" {
		t.Fatalf("multi-source: %+v", s)
	}
	s, err = a.Status(context.Background(), AppRef{"argocd", "missing"})
	if err != nil || s.Found {
		t.Fatalf("missing app: %+v %v", s, err)
	}
	if _, err := New(g, "argocd", nil).Status(context.Background(), AppRef{"argocd", "missing"}); err == nil {
		t.Fatal("without not-found classifier the error must propagate")
	}
	if (AppRef{"a", "b"}).String() != "a/b" {
		t.Fatal("String")
	}
}
