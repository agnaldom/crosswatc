package httpapi

import (
	"testing"

	"github.com/agnaldom/crosswatc/internal/store"
)

func TestDashboardScopedBySquad(t *testing.T) {
	fs, _, h := setup()
	fs.dashboard = store.Dashboard{
		ByState:       map[string]int{"OPERATIONAL": 2, "DEGRADED": 1},
		Health:        store.Health{Ready: 1, NotReady: 1, Unknown: 1},
		ByProvider:    map[string]int{"azure": 3},
		ByEnvironment: map[string]int{"dev": 2, "prd": 1},
		Attention:     []store.AttentionItem{{ID: payID, Name: "pgsql-payments", Kind: "K", State: "DEGRADED", Drift: true}},
		OpenRevisions: []store.OpenRevision{{ResourceID: payID, ResourceName: "pgsql-payments", Rev: 2, State: "VALIDATING"}},
		RecentEvents:  []store.DashboardEvent{{Event: store.Event{ResourceID: &payID, Type: "GITOPS.PR.CREATED"}, ResourceName: "pgsql-payments"}},
	}
	w := do(h, "/api/v1/dashboard", payments)
	if w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	var out dashboardDTO
	decode(t, w, &out)
	if out.Total != 3 || len(out.Attention) != 1 || !out.Attention[0].Drift || len(out.OpenRevisions) != 1 ||
		len(out.RecentEvents) != 1 || out.RecentEvents[0].ResourceName != "pgsql-payments" || out.RecentEvents[0].ResourceID != payID.String() {
		t.Fatalf("%+v", out)
	}
	if out.Health != (healthDTO{Ready: 1, NotReady: 1, Unknown: 1}) || out.ByProvider["azure"] != 3 || out.ByEnvironment["prd"] != 1 {
		t.Fatalf("health/distribution: %+v", out)
	}
	if fs.lastFilter.Owners == nil {
		t.Fatal("squad user must be owner-scoped")
	}
	fs.dashboard = store.Dashboard{}
	decode(t, do(h, "/api/v1/dashboard", map[string]string{"X-Dev-User": "ops", "X-Dev-SRE": "true"}), &out)
	if fs.lastFilter.Owners != nil || out.Attention == nil || out.RecentEvents == nil || out.ByProvider == nil || out.ByState == nil {
		t.Fatalf("SRE global scope and non-null arrays: %+v", out)
	}
}
