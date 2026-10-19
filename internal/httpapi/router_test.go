package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/agnaldom/crosswatc/internal/auth"
	"github.com/agnaldom/crosswatc/internal/correlation"
	"github.com/agnaldom/crosswatc/internal/kube"
	"github.com/agnaldom/crosswatc/internal/store"
)

type fakeStore struct {
	resources    map[uuid.UUID]store.TrackedResource
	projection   map[uuid.UUID]store.Projection
	topology     map[uuid.UUID]store.Topology
	events       []store.Event
	revisions    []store.Revision
	operations   []store.Operation
	lastFilter   store.ResourceFilter
	lastOpFilter store.OperationFilter
	pingErr      error
}

func (f *fakeStore) Ping(context.Context) error { return f.pingErr }
func (f *fakeStore) GetResource(_ context.Context, id uuid.UUID) (store.TrackedResource, error) {
	r, ok := f.resources[id]
	if !ok {
		return r, store.ErrNotFound
	}
	return r, nil
}
func (f *fakeStore) ListResources(_ context.Context, flt store.ResourceFilter, p store.Page) ([]store.TrackedResource, string, error) {
	f.lastFilter = flt
	if p.Cursor == "bad" {
		return nil, "", store.ErrBadCursor
	}
	var out []store.TrackedResource
	for _, r := range f.resources {
		if flt.Owners == nil || contains(flt.Owners, r.Owner) {
			out = append(out, r)
		}
	}
	return out, "next-1", nil
}
func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
func (f *fakeStore) GetProjection(_ context.Context, id uuid.UUID) (store.Projection, error) {
	p, ok := f.projection[id]
	if !ok {
		return p, store.ErrNotFound
	}
	return p, nil
}
func (f *fakeStore) GetTopology(_ context.Context, id uuid.UUID) (store.Topology, error) {
	t, ok := f.topology[id]
	if !ok {
		return t, store.ErrNotFound
	}
	return t, nil
}
func (f *fakeStore) ListEvents(context.Context, uuid.UUID, store.Page) ([]store.Event, string, error) {
	return f.events, "", nil
}
func (f *fakeStore) ListRevisions(context.Context, uuid.UUID) ([]store.Revision, error) {
	return f.revisions, nil
}
func (f *fakeStore) ListRevisionsPage(_ context.Context, _ uuid.UUID, p store.Page) ([]store.Revision, string, error) {
	if p.Cursor == "bad" {
		return nil, "", store.ErrBadCursor
	}
	return f.revisions, "", nil
}
func (f *fakeStore) GetRevision(_ context.Context, id uuid.UUID, rev int) (store.Revision, error) {
	for _, r := range f.revisions {
		if r.ResourceID == id && r.Rev == rev {
			return r, nil
		}
	}
	return store.Revision{}, store.ErrNotFound
}
func (f *fakeStore) ListOperationsPage(_ context.Context, flt store.OperationFilter, p store.Page) ([]store.Operation, string, error) {
	f.lastOpFilter = flt
	if p.Cursor == "bad" {
		return nil, "", store.ErrBadCursor
	}
	var out []store.Operation
	for _, o := range f.operations {
		if (flt.ResourceID == nil || o.ResourceID == *flt.ResourceID) && (flt.RevisionID == nil || (o.RevisionID != nil && *o.RevisionID == *flt.RevisionID)) &&
			(flt.Owners == nil || contains(flt.Owners, f.resources[o.ResourceID].Owner)) {
			out = append(out, o)
		}
	}
	return out, "", nil
}
func (f *fakeStore) GetOperation(_ context.Context, id uuid.UUID) (store.Operation, error) {
	for _, o := range f.operations {
		if o.ID == id {
			o.Owner = f.resources[o.ResourceID].Owner
			return o, nil
		}
	}
	return store.Operation{}, store.ErrNotFound
}
func (f *fakeStore) ListEventsByRevision(_ context.Context, rev uuid.UUID, _ int) ([]store.Event, error) {
	var out []store.Event
	for _, e := range f.events {
		if e.RevisionID != nil && *e.RevisionID == rev {
			out = append(out, e)
		}
	}
	return out, nil
}

type fakeCatalog struct {
	calls int
	err   error
}

func (f *fakeCatalog) DiscoverXRDs(context.Context) ([]kube.XRD, error) {
	f.calls++
	return []kube.XRD{{Group: "database.crosswatc.io", Kind: "XPostgreSQLInstance", ClaimKind: "PostgreSQLInstance", Scope: "Cluster",
		Versions: []kube.XRDVersion{{Name: "v1", Served: true, Referenceable: true, Schema: map[string]any{"type": "object"}}}}}, f.err
}

var (
	payID  = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	cardID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
)

func setup() (*fakeStore, *fakeCatalog, http.Handler) {
	now := time.Date(2026, 10, 17, 10, 0, 0, 0, time.UTC)
	pr := 173
	rev := uuid.New()
	fs := &fakeStore{
		resources: map[uuid.UUID]store.TrackedResource{
			payID:  {ID: payID, Name: "pgsql-payments", Owner: "payments", Kind: "PostgreSQLInstance", ResourceState: "OPERATIONAL", CompositionName: "pg-v3"},
			cardID: {ID: cardID, Name: "pgsql-cards", Owner: "cards", ResourceState: "PROVISIONING"},
		},
		projection: map[uuid.UUID]store.Projection{payID: {
			Stages: []byte(`[{"stage":"REQUEST","status":"SKIPPED"}]`), Health: []byte(`{"argocd":"ok"}`), LastReconcileAt: &now,
		}},
		topology: map[uuid.UUID]store.Topology{payID: {Nodes: []byte(`[{"id":"claim/x"}]`), Edges: []byte(`[]`)}},
		events:   []store.Event{{ID: uuid.New(), ResourceID: &payID, Type: "LIFECYCLE.OPERATIONAL", Source: "crosswatc", Level: "info", CreatedAt: now}},
		revisions: []store.Revision{
			{ID: uuid.New(), ResourceID: payID, Rev: 2, Kind: "UPDATE", State: "SUCCEEDED"},
			{ID: rev, ResourceID: payID, Rev: 1, Kind: "CREATE", State: "SUCCEEDED", PRNumber: &pr, PRURL: "https://gh/pr/173"},
		},
		operations: []store.Operation{{ID: uuid.New(), ResourceID: payID, RevisionID: &rev, Type: "CREATE", Status: "SUCCEEDED", RequestedBy: "agnaldo"}},
	}
	fc := &fakeCatalog{}
	h := New(Deps{Store: fs, Catalog: fc, Auth: auth.Dev{Mapping: auth.GroupMapping{SquadPrefix: "squad-", SREGroup: "sre"}},
		Log: slog.New(slog.NewTextHandler(io.Discard, nil))})
	return fs, fc, h
}

func do(h http.Handler, path string, hdr map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

var payments = map[string]string{"X-Dev-User": "agnaldo", "X-Dev-Squads": "payments"}

func decode(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
		t.Fatalf("decode %q: %v", w.Body.String(), err)
	}
}

func TestAuthAndCorrelation(t *testing.T) {
	_, _, h := setup()
	w := do(h, "/api/v1/resources", nil)
	if w.Code != 401 || w.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("%d %s", w.Code, w.Header().Get("Content-Type"))
	}
	var p Problem
	decode(t, w, &p)
	if p.Type != "https://crosswatc.io/errors/unauthenticated" || !correlation.Valid(p.CorrelationID) || p.CorrelationID != w.Header().Get("X-Correlation-Id") {
		t.Fatalf("%+v", p)
	}
	id := "crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T"
	w = do(h, "/healthz", map[string]string{"X-Correlation-Id": id})
	if w.Code != 200 || w.Header().Get("X-Correlation-Id") != id {
		t.Fatal("inbound correlation id must be echoed")
	}
	if w = do(h, "/healthz", map[string]string{"X-Correlation-Id": "junk"}); !correlation.Valid(w.Header().Get("X-Correlation-Id")) {
		t.Fatal("invalid inbound id must be replaced")
	}
}

func TestListResourcesScopedBySquad(t *testing.T) {
	fs, _, h := setup()
	w := do(h, "/api/v1/resources?environment=dev&provider=azure&kind=K&status=OPERATIONAL&q=pg&owner=payments&limit=10", payments)
	if w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	var pg struct {
		Items      []resourceDTO `json:"items"`
		NextCursor *string       `json:"nextCursor"`
	}
	decode(t, w, &pg)
	if len(pg.Items) != 1 || pg.Items[0].Name != "pgsql-payments" || pg.NextCursor == nil || pg.Items[0].Composition.Name != "pg-v3" {
		t.Fatalf("%+v", pg)
	}
	f := fs.lastFilter
	if f.Environment != "dev" || f.Provider != "azure" || f.Kind != "K" || f.State != "OPERATIONAL" || f.NamePrefix != "pg" || f.Owner != "payments" {
		t.Fatalf("filter %+v", f)
	}

	sre := map[string]string{"X-Dev-User": "ops", "X-Dev-SRE": "true"}
	decode(t, do(h, "/api/v1/resources", sre), &pg)
	if len(pg.Items) != 2 || fs.lastFilter.Owners != nil {
		t.Fatal("SRE has global read")
	}
	if w := do(h, "/api/v1/resources?limit=0", payments); w.Code != 400 {
		t.Fatalf("limit: %d", w.Code)
	}
	if w := do(h, "/api/v1/resources?cursor=bad", payments); w.Code != 400 {
		t.Fatalf("cursor: %d", w.Code)
	}
}

func TestResourceAuthZ(t *testing.T) {
	_, _, h := setup()
	if w := do(h, "/api/v1/resources/"+payID.String(), payments); w.Code != 200 {
		t.Fatalf("own resource: %d", w.Code)
	}
	for _, p := range []string{cardID.String(), uuid.NewString(), "not-a-uuid"} {
		if w := do(h, "/api/v1/resources/"+p+"/state", payments); w.Code != 404 {
			t.Fatalf("%s: %d", p, w.Code)
		}
	}
}

func TestResourceSubresources(t *testing.T) {
	_, _, h := setup()
	base := "/api/v1/resources/" + payID.String()

	var st stateDTO
	decode(t, do(h, base+"/state", payments), &st)
	if st.ResourceState != "OPERATIONAL" || len(st.Stages) != 1 || st.CurrentRevision.Rev != 2 || st.LastReconcileAt == nil {
		t.Fatalf("state %+v", st)
	}

	w := do(h, base+"/health", payments)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"argocd":"ok"`) {
		t.Fatalf("health %s", w.Body)
	}
	w = do(h, base+"/events", payments)
	if !strings.Contains(w.Body.String(), `"type":"LIFECYCLE.OPERATIONAL"`) || !strings.Contains(w.Body.String(), `"nextCursor":null`) {
		t.Fatalf("events %s", w.Body)
	}
	w = do(h, base+"/revisions", payments)
	if !strings.Contains(w.Body.String(), `"prNumber":173`) {
		t.Fatalf("revisions %s", w.Body)
	}
	w = do(h, base+"/operations", payments)
	if !strings.Contains(w.Body.String(), `"requestedBy":"agnaldo"`) {
		t.Fatalf("operations %s", w.Body)
	}
	w = do(h, base+"/topology", payments)
	if !strings.Contains(w.Body.String(), `"nodes":[{"id":"claim/x"}]`) {
		t.Fatalf("topology %s", w.Body)
	}
}

func TestEmptyProjectionAndTopology(t *testing.T) {
	fs, _, h := setup()
	fs.revisions = nil
	sre := map[string]string{"X-Dev-User": "ops", "X-Dev-SRE": "true"}
	base := "/api/v1/resources/" + cardID.String()
	var st stateDTO
	decode(t, do(h, base+"/state", sre), &st)
	if st.CurrentRevision != nil || len(st.Stages) != 0 || string(st.Drift) != `{"detected":false}` {
		t.Fatalf("%+v", st)
	}
	if w := do(h, base+"/topology", sre); w.Body.String() != `{"edges":[],"nodes":[]}` {
		t.Fatalf("topology %s", w.Body)
	}
	if w := do(h, base+"/health", sre); w.Body.String() != `{}` {
		t.Fatalf("health %s", w.Body)
	}
}

func TestCatalog(t *testing.T) {
	_, fc, h := setup()
	w := do(h, "/api/v1/catalog/resource-types", payments)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"claimKind":"PostgreSQLInstance"`) {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if w := do(h, "/api/v1/catalog/resource-types/PostgreSQLInstance", payments); w.Code != 200 {
		t.Fatalf("by claim kind: %d", w.Code)
	}
	w = do(h, "/api/v1/catalog/resource-types/XPostgreSQLInstance/schema", payments)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"openAPIV3Schema":{"type":"object"}`) {
		t.Fatalf("schema %s", w.Body)
	}
	if w := do(h, "/api/v1/catalog/resource-types/Nope", payments); w.Code != 404 {
		t.Fatalf("unknown kind: %d", w.Code)
	}
	if fc.calls != 1 {
		t.Fatalf("catalog must be cached, calls=%d", fc.calls)
	}

	noCatalog := New(Deps{Store: &fakeStore{}, Auth: auth.Dev{}})
	if w := do(noCatalog, "/api/v1/catalog/resource-types", payments); w.Code != 503 {
		t.Fatalf("no catalog: %d", w.Code)
	}
	errCatalog := New(Deps{Store: &fakeStore{}, Catalog: &fakeCatalog{err: errors.New("boom")}, Auth: auth.Dev{}})
	if w := do(errCatalog, "/api/v1/catalog/resource-types", payments); w.Code != 500 {
		t.Fatalf("catalog error: %d", w.Code)
	}
}

func TestReadyAndNotFound(t *testing.T) {
	fs, _, h := setup()
	if w := do(h, "/readyz", nil); w.Code != 200 {
		t.Fatal("ready")
	}
	fs.pingErr = errors.New("down")
	if w := do(h, "/readyz", nil); w.Code != 503 {
		t.Fatal("not ready")
	}
	if w := do(h, "/nope", nil); w.Code != 404 || w.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatal("no route")
	}
}

func TestRevisionHistory(t *testing.T) {
	fs, _, h := setup()
	rev1 := fs.revisions[1]
	fs.events = append(fs.events, store.Event{ID: uuid.New(), ResourceID: &payID, RevisionID: &rev1.ID, Type: "GITOPS.PR.MERGED",
		Source: "github", Level: "info", CreatedAt: time.Now()})
	base := "/api/v1/resources/" + payID.String()

	w := do(h, base+"/revisions/1", payments)
	var d revisionDetailDTO
	decode(t, w, &d)
	if d.Rev != 1 || len(d.Operations) != 1 || d.Operations[0].RequestedBy != "agnaldo" || len(d.Events) != 1 || d.Events[0].Type != "GITOPS.PR.MERGED" {
		t.Fatalf("detail %s", w.Body)
	}
	for _, p := range []string{"/revisions/9", "/revisions/0", "/revisions/abc"} {
		if w := do(h, base+p, payments); w.Code != http.StatusNotFound {
			t.Fatalf("%s: %d", p, w.Code)
		}
	}
	if w := do(h, "/api/v1/resources/"+cardID.String()+"/revisions/1", payments); w.Code != http.StatusNotFound {
		t.Fatalf("foreign resource revision: %d", w.Code)
	}
	if w := do(h, base+"/revisions?cursor=bad", payments); w.Code != http.StatusBadRequest {
		t.Fatalf("bad cursor: %d", w.Code)
	}
}

func TestOperationHistory(t *testing.T) {
	fs, _, h := setup()
	cardOp := store.Operation{ID: uuid.New(), ResourceID: cardID, Type: "UPDATE", Status: "RUNNING", RequestedBy: "bia"}
	fs.operations = append(fs.operations, cardOp)

	w := do(h, "/api/v1/operations?status=SUCCEEDED&type=CREATE", payments)
	var pg page[operationDTO]
	decode(t, w, &pg)
	if len(pg.Items) != 1 || pg.Items[0].ResourceID != payID.String() {
		t.Fatalf("payments sees only its operations: %s", w.Body)
	}
	if f := fs.lastOpFilter; f.Status != "SUCCEEDED" || f.Type != "CREATE" || len(f.Owners) != 1 || f.Owners[0] != "payments" {
		t.Fatalf("filter %+v", f)
	}
	decode(t, do(h, "/api/v1/operations", map[string]string{"X-Dev-User": "ops", "X-Dev-SRE": "true"}), &pg)
	if len(pg.Items) != 2 || fs.lastOpFilter.Owners != nil {
		t.Fatalf("sre sees all: %d %+v", len(pg.Items), fs.lastOpFilter)
	}
	decode(t, do(h, "/api/v1/operations?resourceId="+payID.String(), payments), &pg)
	if fs.lastOpFilter.ResourceID == nil || *fs.lastOpFilter.ResourceID != payID {
		t.Fatalf("resource filter %+v", fs.lastOpFilter)
	}
	decode(t, do(h, "/api/v1/resources/"+payID.String()+"/operations?status=FAILED", payments), &pg)
	if fs.lastOpFilter.ResourceID == nil || fs.lastOpFilter.Status != "FAILED" {
		t.Fatalf("resource operations filter %+v", fs.lastOpFilter)
	}
	for name, path := range map[string]string{
		"bad status":   "/api/v1/operations?status=DONE",
		"bad type":     "/api/v1/operations?type=RESIZE",
		"bad resource": "/api/v1/operations?resourceId=x",
		"bad cursor":   "/api/v1/operations?cursor=bad",
		"bad limit":    "/api/v1/operations?limit=500",
	} {
		if w := do(h, path, payments); w.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d", name, w.Code)
		}
	}

	own := fs.operations[0]
	if w := do(h, "/api/v1/operations/"+own.ID.String(), payments); w.Code != 200 || !strings.Contains(w.Body.String(), own.ID.String()) {
		t.Fatalf("get own: %d %s", w.Code, w.Body)
	}
	for _, id := range []string{cardOp.ID.String(), uuid.NewString(), "nope"} {
		if w := do(h, "/api/v1/operations/"+id, payments); w.Code != http.StatusNotFound {
			t.Fatalf("%s: %d", id, w.Code)
		}
	}
}
