package httpapi

import (
	"net/http"
	"strings"
	"testing"

	"github.com/agnaldom/crosswatc/internal/auth"
	"github.com/agnaldom/crosswatc/internal/changes"
)

func previewHandler() http.Handler {
	fs, _, _ := setup()
	p := fs.projection[payID]
	p.AppliedState = []byte(`{"spec":{"parameters":{"vmSize":"Standard_D2as_v5"}}}`)
	fs.projection[payID] = p
	return New(Deps{Store: fs, Catalog: formCatalog{}, Auth: auth.Dev{Mapping: auth.GroupMapping{SquadPrefix: "squad-", SREGroup: "sre"}}})
}

func TestPreviewEndpoint(t *testing.T) {
	h := previewHandler()
	path := "/api/v1/resources/" + payID.String() + "/changes/preview"

	w := post(h, path, `{"changes":{"vmSize":"Standard_D4as_v5"}}`, payments)
	var res changes.Result
	decode(t, w, &res)
	if w.Code != 200 || !res.Valid || res.Changes[0].From != "Standard_D2as_v5" || res.Changes[0].Impact != changes.ImpactInPlace {
		t.Fatalf("%d %+v", w.Code, res)
	}

	decode(t, post(h, path, `{"changes":{"storage":64}}`, payments), &res)
	if res.Valid || res.Policy.Violations[0].Rule != "not-editable" {
		t.Fatalf("%+v", res)
	}

	sre := map[string]string{"X-Dev-User": "ops", "X-Dev-SRE": "true"}
	if w := post(h, path, `{"changes":{}}`, sre); w.Code != 403 {
		t.Fatalf("SRE must not write: %d", w.Code)
	}
	if w := post(h, path, `{bad`, payments); w.Code != 400 {
		t.Fatalf("bad body: %d", w.Code)
	}
	big := `{"changes":{"vmSize":"` + strings.Repeat("x", maxChangeBody) + `"}}`
	if w := post(h, path, big, payments); w.Code != 400 {
		t.Fatalf("oversized body: %d", w.Code)
	}
}
