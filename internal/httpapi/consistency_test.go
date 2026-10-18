package httpapi

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/agnaldom/crosswatc/internal/auth"
	"github.com/agnaldom/crosswatc/internal/store"
)

// consistencyHandler lets tests control applied state, existing revisions and the Git manifest.
func consistencyHandler(applied string, revs []store.Revision, cs *memChanges, g *fakeGitOps) http.Handler {
	fs, _, _ := setup()
	p := fs.projection[payID]
	p.AppliedState = []byte(applied)
	fs.projection[payID] = p
	fs.revisions = revs
	return New(Deps{Store: fs, Catalog: formCatalog{}, Auth: auth.Dev{Mapping: auth.GroupMapping{SquadPrefix: "squad-"}},
		Changes: cs, GitOps: g})
}

const syncedApplied = `{"syncedWithGit":true,"spec":{"parameters":{"vmSize":"Standard_D2as_v5"}}}`

func TestUpdateBlockedWhenGitDiverges(t *testing.T) {
	path := "/api/v1/resources/" + payID.String() + "/changes"
	g := &fakeGitOps{manifest: "metadata: {name: pgsql-payments}\nspec:\n  parameters:\n    vmSize: Standard_D8as_v5\n"}
	w := post(consistencyHandler(syncedApplied, nil, newMemChanges(), g), path, changeBody, payments)
	if w.Code != 409 || !strings.Contains(w.Body.String(), `"divergences":[{"field":"vmSize","git":"Standard_D8as_v5","cluster":"Standard_D2as_v5"}]`) {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if len(g.reqs) != 0 {
		t.Fatal("no PR may be opened while blocked")
	}

	notSynced := `{"syncedWithGit":false,"spec":{"parameters":{"vmSize":"Standard_D2as_v5"}}}`
	w = post(consistencyHandler(notSynced, nil, newMemChanges(), &fakeGitOps{}), path, changeBody, payments)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "Argo CD") {
		t.Fatalf("%d %s", w.Code, w.Body)
	}

	w = post(consistencyHandler(syncedApplied, nil, newMemChanges(), &fakeGitOps{manifest: "kind: A\n---\nkind: B\n"}), path, changeBody, payments)
	if w.Code != 409 {
		t.Fatalf("unreadable manifest: %d", w.Code)
	}

	if w := post(consistencyHandler(syncedApplied, nil, newMemChanges(), &fakeGitOps{}), path, changeBody, payments); w.Code != 201 {
		t.Fatalf("consistent: %d %s", w.Code, w.Body)
	}
}

func openRev(state string) store.Revision {
	pr := 170
	return store.Revision{ID: uuid.New(), ResourceID: payID, Rev: 4, Kind: "UPDATE", State: state, PRNumber: &pr}
}

func TestOpenRevisionAndSupersede(t *testing.T) {
	path := "/api/v1/resources/" + payID.String() + "/changes"
	open := openRev("WAITING_APPROVAL")

	w := post(consistencyHandler(syncedApplied, []store.Revision{open}, newMemChanges(), &fakeGitOps{}), path, changeBody, payments)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "Revision 4 is WAITING_APPROVAL") {
		t.Fatalf("%d %s", w.Code, w.Body)
	}

	cs, g := newMemChanges(), &fakeGitOps{}
	cs.revs[open.ID] = open
	body := `{"changes":{"vmSize":"Standard_D4as_v5"},"justification":"x","supersede":true}`
	w = post(consistencyHandler(syncedApplied, []store.Revision{open}, cs, g), path, body, payments)
	if w.Code != 201 || g.canceled[0] != 170 || cs.revs[open.ID].State != "SUPERSEDED" {
		t.Fatalf("%d %s %+v", w.Code, w.Body, cs.revs[open.ID])
	}
	var res changeResponse
	decode(t, w, &res)
	if res.Rev != 5 {
		t.Fatalf("new rev %d", res.Rev)
	}

	merged := openRev("APPLYING")
	w = post(consistencyHandler(syncedApplied, []store.Revision{merged}, newMemChanges(), &fakeGitOps{}), path, body, payments)
	if w.Code != 409 {
		t.Fatalf("merged revisions cannot be superseded: %d", w.Code)
	}
}

type revStore struct {
	*fakeStore
	rev store.Revision
}

func (r revStore) GetRevision(context.Context, uuid.UUID, int) (store.Revision, error) {
	if r.rev.ID == uuid.Nil {
		return r.rev, store.ErrNotFound
	}
	return r.rev, nil
}

func TestCancelRevision(t *testing.T) {
	fs, _, _ := setup()
	rev := openRev("VALIDATING")
	cs, g := newMemChanges(), &fakeGitOps{}
	cs.revs[rev.ID] = rev
	cs.ops[uuid.New()] = store.Operation{RevisionID: &rev.ID, Status: "RUNNING"}
	h := New(Deps{Store: revStore{fs, rev}, Catalog: formCatalog{}, Auth: auth.Dev{Mapping: auth.GroupMapping{SquadPrefix: "squad-"}},
		Changes: cs, GitOps: g})
	path := "/api/v1/resources/" + payID.String() + "/revisions/4/cancel"

	if w := post(h, path, `{"reason":"não precisa"}`, payments); w.Code != 204 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if cs.revs[rev.ID].State != "CANCELLED" || g.canceled[0] != 170 {
		t.Fatalf("%+v", cs.revs[rev.ID])
	}
	for _, o := range cs.ops {
		if o.Status != "CANCELLED" {
			t.Fatalf("op %+v", o)
		}
	}
	if w := post(h, path, ``, map[string]string{"X-Dev-User": "x", "X-Dev-Squads": "cards"}); w.Code != 404 {
		t.Fatalf("other squad: %d", w.Code)
	}

	done := openRev("APPLYING")
	h = New(Deps{Store: revStore{fs, done}, Auth: auth.Dev{Mapping: auth.GroupMapping{SquadPrefix: "squad-"}}, Changes: cs, GitOps: g})
	if w := post(h, path, ``, payments); w.Code != 409 {
		t.Fatalf("after merge: %d", w.Code)
	}
	h = New(Deps{Store: revStore{fs, store.Revision{}}, Auth: auth.Dev{Mapping: auth.GroupMapping{SquadPrefix: "squad-"}}, Changes: cs, GitOps: g})
	if w := post(h, path, ``, payments); w.Code != 404 {
		t.Fatalf("missing: %d", w.Code)
	}
}
