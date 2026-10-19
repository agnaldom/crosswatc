package kube

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

func TestEventsFor(t *testing.T) {
	var queries []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Path+"?"+r.URL.Query().Get("fieldSelector"))
		items := []any{
			map[string]any{"type": "Warning", "reason": "CannotCreateExternalResource", "message": "AuthorizationFailed", "count": int64(4),
				"source": map[string]any{"component": "managed/flexibleserver"}, "firstTimestamp": "2026-10-19T10:00:00Z", "lastTimestamp": "2026-10-19T10:05:00Z",
				"metadata": map[string]any{"name": "e1", "creationTimestamp": "2026-10-19T10:00:00Z"}},
			map[string]any{"type": "Normal", "reason": "CreatedExternalResource", "message": "ok", "reportingComponent": "crossplane",
				"eventTime": "2026-10-19T10:06:00.000000Z", "series": map[string]any{"count": int64(2), "lastObservedTime": "2026-10-19T10:07:00.000000Z"},
				"metadata": map[string]any{"name": "e2", "creationTimestamp": "2026-10-19T10:06:00Z"}},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"apiVersion": "v1", "kind": "EventList", "items": items})
	}))
	defer srv.Close()
	dyn, _ := dynamic.NewForConfig(&rest.Config{Host: srv.URL})
	c := &Client{Dynamic: dyn}

	es, err := c.EventsFor(context.Background(), Ref{Kind: "FlexibleServer", Name: "srv"})
	if err != nil {
		t.Fatal(err)
	}
	if queries[0] != "/api/v1/namespaces/default/events?involvedObject.kind=FlexibleServer,involvedObject.name=srv" {
		t.Fatalf("cluster-scoped objects must query the default namespace: %s", queries[0])
	}
	if len(es) != 2 || es[0].Reason != "CreatedExternalResource" || es[0].Count != 2 || es[0].Source != "crossplane" {
		t.Fatalf("newest first, series count: %+v", es)
	}
	if w := es[1]; w.Type != "Warning" || w.Count != 4 || w.Source != "managed/flexibleserver" || w.LastSeen.Minute() != 5 {
		t.Fatalf("%+v", w)
	}

	if _, err := c.EventsFor(context.Background(), Ref{Kind: "PostgreSQLInstance", Name: "pg", Namespace: "team"}); err != nil {
		t.Fatal(err)
	}
	if queries[1] != "/api/v1/namespaces/team/events?involvedObject.kind=PostgreSQLInstance,involvedObject.name=pg,involvedObject.namespace=team" {
		t.Fatalf("%s", queries[1])
	}
}
