package httpapi

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/agnaldom/crosswatc/internal/auth"
	"github.com/agnaldom/crosswatc/internal/gitops"
	"github.com/agnaldom/crosswatc/internal/store"
)

type memChanges struct {
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
func (m *memChanges) InsertEvent(_ context.Context, e store.Event) (bool, error) {
	m.events = append(m.events, e)
	return true, nil
}

type fakeGitOps struct {
	err  error
	reqs []gitops.Request
}

func (f *fakeGitOps) Submit(_ context.Context, r gitops.Request) (gitops.Result, error) {
	f.reqs = append(f.reqs, r)
	return gitops.Result{Branch: "update/x/dev/y/rev-3", CommitSHA: "c1", PRNumber: 173, PRURL: "https://gh/pr/173"}, f.err
}
func (f *fakeGitOps) Manifest(context.Context, string, string) ([]byte, error) { return nil, nil }
func (f *fakeGitOps) Cancel(context.Context, string, int, string, string) error {
	return nil
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
