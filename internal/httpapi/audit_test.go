package httpapi

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/agnaldom/crosswatc/internal/auth"
	"github.com/agnaldom/crosswatc/internal/store"
)

type fakeAudit struct {
	last  store.AuditFilter
	items []store.AuditEntry
	err   error
}

func (f *fakeAudit) ListAudit(_ context.Context, flt store.AuditFilter, _ store.Page) ([]store.AuditEntry, string, error) {
	f.last = flt
	return f.items, "", f.err
}

func TestAuditWrittenOnChangeAndCancel(t *testing.T) {
	cs, g := newMemChanges(), &fakeGitOps{}
	h := changeHandler(cs, g)
	hdr := map[string]string{"X-Dev-User": "agnaldo", "X-Dev-Squads": "payments"}
	if w := post(h, "/api/v1/resources/"+payID.String()+"/changes", changeBody, hdr); w.Code != 201 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	a := cs.audits[0]
	if a.Actor != "agnaldo" || a.Action != AuditChangeRequested || string(a.Before) != `{"vmSize":"Standard_D2as_v5"}` ||
		string(a.After) != `{"vmSize":"Standard_D4as_v5"}` || a.PRURL == "" || a.OperationID == nil || a.RevisionID == nil {
		t.Fatalf("%+v", a)
	}

	cs = newMemChanges()
	if w := post(changeHandler(cs, &fakeGitOps{err: errors.New("x")}), "/api/v1/resources/"+payID.String()+"/changes", changeBody, hdr); w.Code != 502 {
		t.Fatal(w.Code)
	}
	if cs.audits[0].Action != AuditChangeFailed {
		t.Fatalf("%+v", cs.audits)
	}

	fs, _, _ := setup()
	rev := openRev("VALIDATING")
	cs = newMemChanges()
	h = New(Deps{Store: revStore{fs, rev}, Auth: auth.Dev{Mapping: auth.GroupMapping{SquadPrefix: "squad-"}}, Changes: cs, GitOps: &fakeGitOps{}})
	if w := post(h, "/api/v1/resources/"+payID.String()+"/revisions/4/cancel", `{"reason":"duplicada"}`, hdr); w.Code != 204 {
		t.Fatal(w.Code)
	}
	if cs.audits[0].Action != AuditRevisionCancelled || !strings.Contains(string(cs.audits[0].After), "duplicada") {
		t.Fatalf("%+v", cs.audits)
	}
}

func TestAuditEndpoints(t *testing.T) {
	fs, _, _ := setup()
	rid := payID
	fa := &fakeAudit{items: []store.AuditEntry{{ID: 7, Actor: "agnaldo", Action: AuditChangeRequested, ResourceID: &rid,
		After: []byte(`{"vmSize":"D4"}`)}}}
	h := New(Deps{Store: fs, Auth: auth.Dev{Mapping: auth.GroupMapping{SquadPrefix: "squad-", SREGroup: "sre"}}, Audit: fa})

	w := do(h, "/api/v1/audit?actor=agnaldo", payments)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"action":"change.requested"`) || !strings.Contains(w.Body.String(), `"after":{"vmSize":"D4"}`) {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if fa.last.Actor != "agnaldo" || len(fa.last.Owners) != 1 || fa.last.Owners[0] != "payments" {
		t.Fatalf("scope %+v", fa.last)
	}
	do(h, "/api/v1/audit", map[string]string{"X-Dev-User": "ops", "X-Dev-SRE": "true"})
	if fa.last.Owners != nil {
		t.Fatal("SRE sees everything")
	}
	if w := do(h, "/api/v1/audit?resourceId=x", payments); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if w := do(h, "/api/v1/resources/"+payID.String()+"/audit", payments); w.Code != 200 || *fa.last.ResourceID != payID {
		t.Fatal(w.Code)
	}
	if w := do(h, "/api/v1/resources/"+cardID.String()+"/audit", payments); w.Code != 404 {
		t.Fatal("other squad's audit is hidden")
	}
	if w := do(New(Deps{Store: fs, Auth: auth.Dev{}}), "/api/v1/audit", payments); w.Code != 503 {
		t.Fatal(w.Code)
	}
}
