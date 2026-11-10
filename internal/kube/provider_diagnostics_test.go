package kube

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"k8s.io/client-go/rest"
)

func TestProviderDiagnosticsAndScopedLogs(t *testing.T) {
	requests := []string{}
	provider := map[string]any{"apiVersion": "pkg.crossplane.io/v1", "kind": "Provider", "metadata": map[string]any{"name": "provider-a", "uid": "provider-uid"}, "spec": map[string]any{"package": "xpkg.example/provider-a:v1.0.0"}, "status": map[string]any{"currentRevision": "rev-a", "conditions": []any{map[string]any{"type": "Healthy", "status": "Unknown", "message": "missing dependencies"}}}}
	revision := map[string]any{"apiVersion": "pkg.crossplane.io/v1", "kind": "ProviderRevision", "metadata": map[string]any{"name": "rev-a", "uid": "revision-uid", "ownerReferences": []any{map[string]any{"apiVersion": "pkg.crossplane.io/v1", "kind": "Provider", "name": "provider-a", "uid": "provider-uid"}}}, "status": map[string]any{"conditions": []any{map[string]any{"type": "Healthy", "status": "False", "message": "dependency not installed"}}}}
	pod := func(name, ns, provider, rev string) map[string]any {
		return map[string]any{"apiVersion": "v1", "kind": "Pod", "metadata": map[string]any{"name": name, "namespace": ns, "labels": map[string]any{"pkg.crossplane.io/provider": provider, "pkg.crossplane.io/revision": rev}}, "spec": map[string]any{"containers": []any{map[string]any{"name": "package-runtime"}}}, "status": map[string]any{"phase": "Pending", "containerStatuses": []any{map[string]any{"name": "package-runtime", "restartCount": int64(1), "ready": false, "state": map[string]any{"waiting": map[string]any{"reason": "CrashLoopBackOff", "message": "process exited"}}}}}}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path+"?"+r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		write := func(v any) { _ = json.NewEncoder(w).Encode(v) }
		switch r.URL.Path {
		case "/apis/pkg.crossplane.io/v1/providers/provider-a":
			write(provider)
		case "/apis/pkg.crossplane.io/v1/providerrevisions/rev-a":
			write(revision)
		case "/api/v1/namespaces/default/events":
			if !strings.HasPrefix(r.URL.Query().Get("fieldSelector"), "involvedObject.uid=") {
				t.Error("events must be scoped by UID")
			}
			write(map[string]any{"apiVersion": "v1", "kind": "EventList", "items": []any{map[string]any{"metadata": map[string]any{"name": "warning"}, "type": "Warning", "reason": "ResolveDependencies", "message": "family is missing"}}})
		case "/api/v1/namespaces/crossplane-system/pods":
			if r.URL.Query().Get("labelSelector") == "app=crossplane" {
				p := pod("crossplane", "crossplane-system", "", "")
				p["metadata"].(map[string]any)["labels"] = map[string]any{"app": "crossplane"}
				write(map[string]any{"apiVersion": "v1", "kind": "PodList", "items": []any{p}})
			} else {
				write(map[string]any{"apiVersion": "v1", "kind": "PodList", "items": []any{pod("runtime-a", "crossplane-system", "provider-a", "rev-a"), pod("wrong-revision", "crossplane-system", "provider-a", "old"), pod("unrelated", "kube-system", "provider-a", "rev-a"), pod("wrong-provider", "crossplane-system", "other", "rev-a")}})
			}
		case "/api/v1/namespaces/crossplane-system/pods/runtime-a/log":
			if r.URL.Query().Get("tailLines") != "200" || r.URL.Query().Get("limitBytes") != "65536" {
				t.Error("logs must be bounded")
			}
			w.Header().Set("Content-Type", "text/plain")
			if r.URL.Query().Get("previous") != "true" {
				w.WriteHeader(400)
				_, _ = w.Write([]byte("container not running"))
				return
			}
			_, _ = w.Write([]byte("previous runtime crash"))
		case "/api/v1/namespaces/crossplane-system/pods/crossplane/log":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("provider-a dependency failed\nprovider-ab unrelated secret\nother-provider unrelated\ncontroller: leader election lost\n"))
		default:
			t.Errorf("unexpected access: %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, err := New(&rest.Config{Host: server.URL}, 100)
	if err != nil {
		t.Fatal(err)
	}
	diagnostics, err := client.ProviderDiagnostics(context.Background(), "provider-a")
	if err != nil || len(diagnostics.Pods) != 1 || len(diagnostics.RevisionConditions) != 1 || len(diagnostics.Events) != 2 || diagnostics.Pods[0].Containers[0].Reason != "CrashLoopBackOff" {
		t.Fatalf("%+v %v", diagnostics, err)
	}
	logs, err := client.ProviderLogs(context.Background(), "provider-a")
	if err != nil || len(logs) != 3 || !logs[0].Previous || logs[0].Text != "previous runtime crash" || logs[1].Text != "provider-a dependency failed\ncontroller: leader election lost" || !logs[2].Previous {
		t.Fatalf("%+v %v", logs, err)
	}
	for _, request := range requests {
		if strings.Contains(request, "/kube-system/") || strings.Contains(request, "/unrelated/") || strings.Contains(request, "/wrong-revision/") {
			t.Fatal("unrelated logs were accessed: " + request)
		}
	}
	// A stale/malicious revision pointer must not expose another Provider's conditions.
	revision["metadata"].(map[string]any)["ownerReferences"] = []any{map[string]any{"kind": "Provider", "name": "other", "uid": "other"}}
	diagnostics, err = client.ProviderDiagnostics(context.Background(), "provider-a")
	if err != nil || len(diagnostics.RevisionConditions) != 0 || len(diagnostics.Warnings) == 0 {
		t.Fatalf("foreign revision: %+v %v", diagnostics, err)
	}
}
