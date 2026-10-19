package httpapi

import (
	"net/http"
	"strings"
	"testing"

	"github.com/agnaldom/crosswatc/internal/auth"
	"github.com/agnaldom/crosswatc/internal/store"
)

func decomHandler(envs []string, revs []store.Revision, cs *memChanges, g *fakeGitOps, mut func(*fakeStore)) http.Handler {
	fs, _, _ := setup()
	r := fs.resources[payID]
	r.Environment, r.Repository, r.GitPath, r.ClaimName = "dev", "org/argocd", "claims/dev/pg.yaml", "pgsql-payments"
	fs.resources[payID] = r
	fs.topology[payID] = store.Topology{Nodes: []byte(`[
		{"id":"c","type":"claim","kind":"PostgreSQLInstance","name":"pgsql-payments","status":"ready"},
		{"id":"m1","type":"managed-resource","kind":"FlexibleServer","name":"srv","status":"ready","provider":"azure","deletionPolicy":"Delete"},
		{"id":"m2","type":"managed-resource","kind":"ResourceGroup","name":"rg","status":"ready","provider":"azure","deletionPolicy":"Orphan"}]`),
		Edges: []byte(`[]`)}
	fs.revisions = revs
	if mut != nil {
		mut(fs)
	}
	d := Deps{Store: fs, Auth: auth.Dev{Mapping: auth.GroupMapping{SquadPrefix: "squad-", SREGroup: "sre"}}, DecommissionEnvironments: envs}
	if cs != nil {
		d.Changes, d.GitOps = cs, g
	}
	return New(d)
}

const decomBody = `{"confirmName":"pgsql-payments","justification":"Serviço migrado para outro banco"}`

func TestDecommissionPreview(t *testing.T) {
	h := decomHandler([]string{"dev"}, nil, nil, nil, nil)
	w := post(h, "/api/v1/resources/"+payID.String()+"/decommission/preview", ``, payments)
	var p decommissionPreview
	decode(t, w, &p)
	if w.Code != 200 || !p.Allowed || len(p.Deleted) != 1 || p.Deleted[0].Name != "srv" || len(p.Orphaned) != 1 || p.ConfirmName != "pgsql-payments" {
		t.Fatalf("%d %+v", w.Code, p)
	}
	decode(t, post(decomHandler([]string{"hlg"}, nil, nil, nil, nil), "/api/v1/resources/"+payID.String()+"/decommission/preview", ``, payments), &p)
	if p.Allowed || !strings.Contains(p.Reason, "não é permitido") {
		t.Fatalf("%+v", p)
	}
	sre := map[string]string{"X-Dev-User": "ops", "X-Dev-SRE": "true"}
	if w := post(h, "/api/v1/resources/"+payID.String()+"/decommission/preview", ``, sre); w.Code != 403 {
		t.Fatalf("SRE cannot decommission: %d", w.Code)
	}
}

func TestDecommission(t *testing.T) {
	path := "/api/v1/resources/" + payID.String() + "/decommission"
	cs, g := newMemChanges(), &fakeGitOps{}
	w := post(decomHandler([]string{"*"}, nil, cs, g, nil), path, decomBody, payments)
	if w.Code != 201 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if len(g.deletes) != 1 || g.deletes[0].Deleted[0] != "FlexibleServer/srv" || g.deletes[0].Orphaned[0] != "ResourceGroup/rg" {
		t.Fatalf("%+v", g.deletes)
	}
	for _, rv := range cs.revs {
		if rv.Kind != "DELETE" || rv.State != "VALIDATING" || *rv.PRNumber != 181 {
			t.Fatalf("%+v", rv)
		}
	}
	for _, o := range cs.ops {
		if o.Type != "DELETE" || o.Status != "RUNNING" {
			t.Fatalf("%+v", o)
		}
	}
	if cs.audits[0].Action != AuditDecommissionRequested {
		t.Fatalf("%+v", cs.audits)
	}

	cases := map[string]struct {
		envs []string
		body string
		revs []store.Revision
		mut  func(*fakeStore)
		code int
	}{
		"wrong confirmation":  {[]string{"*"}, `{"confirmName":"pgsql","justification":"Serviço migrado para outro banco"}`, nil, nil, 400},
		"short justification": {[]string{"*"}, `{"confirmName":"pgsql-payments","justification":"x"}`, nil, nil, 400},
		"env not allowed":     {[]string{"hlg"}, decomBody, nil, nil, 409},
		"disabled":            {nil, decomBody, nil, nil, 409},
		"open revision":       {[]string{"*"}, decomBody, []store.Revision{openRev("WAITING_APPROVAL")}, nil, 409},
		"already deleting": {[]string{"*"}, decomBody, nil, func(fs *fakeStore) {
			r := fs.resources[payID]
			r.ResourceState = "DELETE_PENDING"
			fs.resources[payID] = r
		}, 409},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if w := post(decomHandler(tc.envs, tc.revs, newMemChanges(), &fakeGitOps{}, tc.mut), path, tc.body, payments); w.Code != tc.code {
				t.Fatalf("%d %s", w.Code, w.Body)
			}
		})
	}
	if w := post(decomHandler([]string{"*"}, nil, nil, nil, nil), path, decomBody, payments); w.Code != 503 {
		t.Fatalf("no gitops: %d", w.Code)
	}
}

// #82: production is an explicit set, not a "pr" prefix.
func TestDecommissionProductionWarning(t *testing.T) {
	for env, want := range map[string]bool{"preview": false, "pre": false, "prd": true, "prod": true, "production": true, "live": false} {
		h := decomHandler([]string{"*"}, nil, nil, nil, func(fs *fakeStore) {
			r := fs.resources[payID]
			r.Environment = env
			fs.resources[payID] = r
		})
		var p decommissionPreview
		decode(t, post(h, "/api/v1/resources/"+payID.String()+"/decommission/preview", ``, payments), &p)
		got := false
		for _, w := range p.Warnings {
			if strings.Contains(w, "Ambiente produtivo") {
				got = true
			}
		}
		if got != want {
			t.Errorf("%s: production warning %v, want %v", env, got, want)
		}
	}
	if productionEnvs([]string{"live"})[0] != "live" {
		t.Fatal("configured list overrides defaults")
	}
}
