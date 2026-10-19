package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/agnaldom/crosswatc/internal/forms"
	"github.com/agnaldom/crosswatc/internal/kube"

	"github.com/google/uuid"

	"github.com/agnaldom/crosswatc/internal/auth"
	"github.com/agnaldom/crosswatc/internal/gitops"
	"github.com/agnaldom/crosswatc/internal/store"
)

type memChanges struct {
	audits   []store.AuditEntry
	revs     map[uuid.UUID]store.Revision
	ops      map[uuid.UUID]store.Operation
	events   []store.Event
	conflict bool
}

func newMemChanges() *memChanges {
	return &memChanges{revs: map[uuid.UUID]store.Revision{}, ops: map[uuid.UUID]store.Operation{}}
}
func (m *memChanges) InsertRevision(_ context.Context, r store.Revision) (store.Revision, error) {
	if m.conflict {
		return r, store.ErrConflict
	}
	r.ID = uuid.New()
	m.revs[r.ID] = r
	return r, nil
}
func (m *memChanges) UpdateRevision(_ context.Context, r store.Revision) error {
	m.revs[r.ID] = r
	return nil
}
func (m *memChanges) InsertOperation(_ context.Context, o store.Operation) (store.Operation, error) {
	o.ID = uuid.New()
	m.ops[o.ID] = o
	return o, nil
}
func (m *memChanges) UpdateOperationsForRevision(_ context.Context, id uuid.UUID, status string, _ *time.Time) error {
	for k, o := range m.ops {
		if o.RevisionID != nil && *o.RevisionID == id {
			o.Status = status
			m.ops[k] = o
		}
	}
	return nil
}
func (m *memChanges) InsertAudit(_ context.Context, a store.AuditEntry) error {
	m.audits = append(m.audits, a)
	return nil
}
func (m *memChanges) InsertEvent(_ context.Context, e store.Event) (bool, error) {
	m.events = append(m.events, e)
	return true, nil
}

type fakeGitOps struct {
	err      error
	reqs     []gitops.Request
	manifest string
	canceled []int
	deletes  []gitops.DeleteRequest
}

func (f *fakeGitOps) Submit(_ context.Context, r gitops.Request) (gitops.Result, error) {
	f.reqs = append(f.reqs, r)
	return gitops.Result{Branch: "update/x/dev/y/rev-3", CommitSHA: "c1", PRNumber: 173, PRURL: "https://gh/pr/173"}, f.err
}
func (f *fakeGitOps) SubmitDelete(_ context.Context, r gitops.DeleteRequest) (gitops.Result, error) {
	f.deletes = append(f.deletes, r)
	return gitops.Result{Branch: "delete/x", CommitSHA: "c2", PRNumber: 181, PRURL: "https://gh/pr/181"}, f.err
}
func (f *fakeGitOps) Manifest(context.Context, string, string) ([]byte, error) {
	if f.manifest == "" {
		return []byte("metadata: {name: pgsql-payments}\nspec:\n  parameters:\n    vmSize: Standard_D2as_v5\n"), nil
	}
	return []byte(f.manifest), nil
}
func (f *fakeGitOps) Cancel(_ context.Context, _ string, n int, _, _ string) error {
	f.canceled = append(f.canceled, n)
	return f.err
}

func changeHandler(cs *memChanges, g *fakeGitOps) http.Handler {
	fs, _, _ := setup()
	p := fs.projection[payID]
	p.AppliedState = []byte(`{"spec":{"parameters":{"vmSize":"Standard_D2as_v5"}}}`)
	fs.projection[payID] = p
	d := Deps{Store: fs, Catalog: formCatalog{}, Auth: auth.Dev{Mapping: auth.GroupMapping{SquadPrefix: "squad-"}}}
	if cs != nil {
		d.Changes, d.GitOps = cs, g
	}
	return New(d)
}

const changeBody = `{"changes":{"vmSize":"Standard_D4as_v5"},"justification":"Aumento de capacidade."}`

func TestCreateChange(t *testing.T) {
	cs, g := newMemChanges(), &fakeGitOps{}
	h := changeHandler(cs, g)
	path := "/api/v1/resources/" + payID.String() + "/changes"
	hdr := map[string]string{"X-Dev-User": "agnaldo", "X-Dev-Squads": "payments", "X-Correlation-Id": "crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T"}

	w := post(h, path, changeBody, hdr)
	var res changeResponse
	decode(t, w, &res)
	if w.Code != 201 || res.Rev != 3 || res.PRNumber != 173 || res.CorrelationID != "crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T" {
		t.Fatalf("%d %+v", w.Code, res)
	}
	rev := cs.revs[uuid.MustParse(res.RevisionID)]
	if rev.State != "VALIDATING" || rev.Kind != "UPDATE" || *rev.PRNumber != 173 || rev.Branch == "" || len(rev.Changes) == 0 {
		t.Fatalf("revision %+v", rev)
	}
	op := cs.ops[uuid.MustParse(res.OperationID)]
	if op.Status != "RUNNING" || op.RequestedBy != "agnaldo" || op.Type != "UPDATE" {
		t.Fatalf("operation %+v", op)
	}
	gr := g.reqs[0]
	if gr.Rev != 3 || gr.Changes[0].To != "Standard_D4as_v5" || gr.Justification == "" {
		t.Fatalf("gitops req %+v", gr)
	}
	if len(cs.events) != 2 || cs.events[1].Type != "GITOPS.PR.CREATED" {
		t.Fatalf("events %+v", cs.events)
	}
}

func TestCreateChangeErrors(t *testing.T) {
	path := "/api/v1/resources/" + payID.String() + "/changes"

	if w := post(changeHandler(nil, nil), path, changeBody, payments); w.Code != 503 {
		t.Fatalf("no gitops: %d", w.Code)
	}
	h := changeHandler(newMemChanges(), &fakeGitOps{})
	if w := post(h, path, `{"changes":{"vmSize":"Standard_D4as_v5"}}`, payments); w.Code != 400 {
		t.Fatalf("missing justification: %d", w.Code)
	}
	if w := post(h, path, `{"changes":{"storage":1},"justification":"x"}`, payments); w.Code != 422 {
		t.Fatalf("invalid change: %d", w.Code)
	}

	cs := newMemChanges()
	cs.conflict = true
	if w := post(changeHandler(cs, &fakeGitOps{}), path, changeBody, payments); w.Code != 409 {
		t.Fatalf("open revision: %d", w.Code)
	}

	cs = newMemChanges()
	w := post(changeHandler(cs, &fakeGitOps{err: errors.New("boom")}), path, changeBody, payments)
	if w.Code != 502 {
		t.Fatalf("gitops failure: %d", w.Code)
	}
	for _, r := range cs.revs {
		if r.State != "FAILED" || len(r.FailureReason) == 0 {
			t.Fatalf("revision must be FAILED: %+v", r)
		}
	}
	for _, o := range cs.ops {
		if o.Status != "FAILED" {
			t.Fatalf("operation must be FAILED: %+v", o)
		}
	}
}

// #79: new values of sensitive fields are never persisted or echoed.
func TestSensitiveValuesNotPersisted(t *testing.T) {
	fs, _, _ := setup()
	p := fs.projection[payID]
	p.AppliedState = []byte(`{"spec":{"parameters":{"adminPassword":"[REDACTED]"}}}`)
	fs.projection[payID] = p
	cs, g := newMemChanges(), &fakeGitOps{manifest: "metadata: {name: pgsql-payments}\nspec:\n  parameters: {}\n"}
	h := New(Deps{Store: fs, Catalog: secretCatalog{}, Auth: auth.Dev{Mapping: auth.GroupMapping{SquadPrefix: "squad-"}}, Changes: cs, GitOps: g})
	w := post(h, "/api/v1/resources/"+payID.String()+"/changes/preview", `{"changes":{"adminPassword":"n3w-s3cret"}}`, payments)
	if strings.Contains(w.Body.String(), "n3w-s3cret") {
		t.Fatalf("preview echoed the secret: %s", w.Body)
	}
	w = post(h, "/api/v1/resources/"+payID.String()+"/changes", `{"changes":{"adminPassword":"n3w-s3cret"},"justification":"rotação de senha"}`, payments)
	if w.Code != 201 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if g.reqs[0].Changes[0].To != "n3w-s3cret" {
		t.Fatal("the manifest edit still needs the real value")
	}
	for _, rv := range cs.revs {
		if strings.Contains(string(rv.Changes), "n3w-s3cret") {
			t.Fatalf("revision.changes: %s", rv.Changes)
		}
	}
	for _, a := range cs.audits {
		if strings.Contains(string(a.After), "n3w-s3cret") || strings.Contains(string(a.Before), "n3w-s3cret") {
			t.Fatalf("audit: %s", a.After)
		}
	}
}

type secretCatalog struct{}

func (secretCatalog) DiscoverXRDs(context.Context) ([]kube.XRD, error) {
	x := pgXRDWithForm()
	x.Annotations = map[string]string{forms.MetadataAnnotation: `{"adminPassword": {"editable": true}}`}
	x.Versions[0].Schema = map[string]any{"type": "object", "properties": map[string]any{"spec": map[string]any{"type": "object", "properties": map[string]any{
		"parameters": map[string]any{"type": "object", "properties": map[string]any{"adminPassword": map[string]any{"type": "string"}}}}}}}
	return []kube.XRD{x}, nil
}
