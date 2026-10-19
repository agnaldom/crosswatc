package httpapi

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agnaldom/crosswatc/internal/auth"
	"github.com/agnaldom/crosswatc/internal/kube"
	"github.com/agnaldom/crosswatc/internal/store"
)

type fakeCluster struct {
	mu   sync.Mutex
	refs []kube.Ref
}

func (f *fakeCluster) EventsFor(_ context.Context, ref kube.Ref) ([]kube.Event, error) {
	f.mu.Lock()
	f.refs = append(f.refs, ref)
	f.mu.Unlock()
	if ref.Name == "broken" {
		return nil, errors.New("forbidden")
	}
	return []kube.Event{{Object: ref, Type: "Warning", Reason: "R-" + ref.Name, LastSeen: time.Date(2026, 1, 1, 0, len(ref.Name), 0, 0, time.UTC)}}, nil
}

func TestKubeEvents(t *testing.T) {
	fs, _, _ := setup()
	fs.topology[payID] = store.Topology{Nodes: []byte(`[
		{"id":"c","type":"claim","apiVersion":"db/v1","kind":"PostgreSQLInstance","name":"pg","namespace":"team","status":"ready"},
		{"id":"x","type":"composite","apiVersion":"db/v1","kind":"XPostgreSQLInstance","name":"xpg-long","status":"ready"},
		{"id":"m","type":"managed-resource","apiVersion":"az/v1","kind":"Server","name":"broken","status":"error"},
		{"id":"g","type":"managed-resource","kind":"Vnet","name":"gone","status":"missing"}]`), Edges: []byte(`[]`)}
	fc := &fakeCluster{}
	h := New(Deps{Store: fs, Cluster: fc, Auth: auth.Dev{Mapping: auth.GroupMapping{SquadPrefix: "squad-"}}})
	w := do(h, "/api/v1/resources/"+payID.String()+"/kube-events", payments)
	if w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"reason":"R-xpg-long"`) || !strings.Contains(body, `"error":"forbidden"`) {
		t.Fatalf("%s", body)
	}
	if strings.Index(body, "R-xpg-long") > strings.Index(body, `"R-pg"`) {
		t.Fatal("newest first")
	}
	if len(fc.refs) != 3 {
		t.Fatalf("missing nodes must not be queried: %+v", fc.refs)
	}
	if w := do(h, "/api/v1/resources/"+cardID.String()+"/kube-events", payments); w.Code != 404 {
		t.Fatal("other squad's events are hidden")
	}
	if w := do(New(Deps{Store: fs, Auth: auth.Dev{Mapping: auth.GroupMapping{SquadPrefix: "squad-"}}}), "/api/v1/resources/"+payID.String()+"/kube-events", payments); w.Code != 503 {
		t.Fatal(w.Code)
	}
}
