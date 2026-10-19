package events

import (
	"encoding/json"
	"slices"
	"testing"
	"time"
)

const reqID = "crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T"

var now = time.Date(2026, 10, 17, 10, 0, 0, 0, time.UTC)

func TestCatalogCoversSpec(t *testing.T) {
	// every type of §33 has a level and a status
	for _, typ := range []Type{RequestCreated, RequestScaffolderCompleted, RequestScaffolderFailed, GitOpsPRCreated, GitOpsPRClosed,
		GitOpsCheckSucceeded, GitOpsPRApproved, GitOpsPRMerged, GitOpsCheckFailed, GitOpsSyncStarted, GitOpsSyncSucceeded,
		GitOpsSyncFailed, InfraClaimCreated, InfraClaimDeleted, InfraClaimInvalidMetadata, InfraXRCreated, InfraReconcileStarted,
		InfraReconcileReady, InfraReconcileFailed, InfraManagedResourceCreated, InfraManagedResourceReady, InfraManagedResourceFailed,
		LifecycleOperational, LifecycleDegraded, LifecycleDriftDetect, LifecycleDriftResolve, OperationStuck} {
		if !Known(typ) {
			t.Errorf("%s missing from catalog", typ)
		}
		if err := New("id", SourceCrosswatc, typ, now).Validate(); err != nil {
			t.Errorf("%s: %v", typ, err)
		}
	}
	if DefaultLevel(GitOpsSyncFailed) != Error || DefaultStatus(GitOpsSyncStarted) != InProgress || DefaultLevel(OperationStuck) != Warning {
		t.Fatal("unexpected catalog values")
	}
}

func TestValidate(t *testing.T) {
	ok := New("e1", SourceGitHub, GitOpsPRMerged, now)
	for name, mut := range map[string]func(*Event){
		"no id":     func(e *Event) { e.EventID = "" },
		"no source": func(e *Event) { e.Source = "" },
		"bad type":  func(e *Event) { e.Type = "PR.MERGED" },
		"bad level": func(e *Event) { e.Level = "debug" },
		"bad stat":  func(e *Event) { e.Status = "done" },
	} {
		e := ok
		mut(&e)
		if e.Validate() == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	b, _ := json.Marshal(ok)
	var back map[string]any
	_ = json.Unmarshal(b, &back)
	for _, k := range []string{"eventId", "source", "type", "status", "timestamp", "level"} {
		if _, found := back[k]; !found {
			t.Errorf("JSON field %s missing (§34)", k)
		}
	}
}

func types(n Normalized) []Type {
	var out []Type
	for _, e := range n.Events {
		out = append(out, e.Type)
	}
	return out
}

func TestFromGitHub(t *testing.T) {
	pr := func(action string, merged bool) string {
		b, _ := json.Marshal(map[string]any{"action": action, "repository": map[string]any{"full_name": "org/gitops"},
			"pull_request": map[string]any{"number": 173, "html_url": "https://gh/pr/173", "title": "Resize VM",
				"body": "Requested via Crosswatc\nRequest: " + reqID, "merged": merged, "head": map[string]any{"ref": "crosswatc/resize"}}})
		return string(b)
	}
	cases := []struct {
		event, body string
		want        []Type
	}{
		{"pull_request", pr("opened", false), []Type{GitOpsPRCreated}},
		{"pull_request", pr("reopened", false), []Type{GitOpsPRCreated}},
		{"pull_request", pr("closed", true), []Type{GitOpsPRMerged}},
		{"pull_request", pr("closed", false), []Type{GitOpsPRClosed}},
		{"pull_request", pr("synchronize", false), nil},
		{"pull_request_review", `{"action":"submitted","review":{"state":"APPROVED"},"pull_request":{"number":173,"title":"` + reqID + `"},"repository":{"full_name":"org/gitops"}}`, []Type{GitOpsPRApproved}},
		{"pull_request_review", `{"action":"submitted","review":{"state":"commented"},"pull_request":{"number":173},"repository":{"full_name":"org/gitops"}}`, nil},
		{"check_run", `{"action":"completed","check_run":{"name":"policy","conclusion":"success","pull_requests":[{"number":173,"head":{"ref":"x-` + reqID + `"}}]},"repository":{"full_name":"org/gitops"}}`, []Type{GitOpsCheckSucceeded}},
		{"check_run", `{"action":"completed","check_run":{"name":"policy","conclusion":"failure"},"repository":{"full_name":"org/gitops"}}`, []Type{GitOpsCheckFailed}},
		{"check_run", `{"action":"created","check_run":{"status":"queued"},"repository":{"full_name":"org/gitops"}}`, nil},
		{"push", `{"ref":"refs/heads/main","head_commit":{"message":"Merge ` + reqID + `"},"repository":{"full_name":"org/gitops"}}`, nil},
	}
	for _, tc := range cases {
		n, err := FromGitHub(tc.event, "del-1", []byte(tc.body), now)
		if err != nil {
			t.Fatalf("%s: %v", tc.event, err)
		}
		if got := types(n); !slices.Equal(got, tc.want) {
			t.Errorf("%s %s: got %v want %v", tc.event, tc.body[:min(40, len(tc.body))], got, tc.want)
		}
		if n.Hints.Repository != "org/gitops" {
			t.Errorf("repository hint %q", n.Hints.Repository)
		}
		for _, e := range n.Events {
			if err := e.Validate(); err != nil || e.Source != SourceGitHub || e.EventID == "" {
				t.Errorf("invalid event %+v: %v", e, err)
			}
		}
	}
	n, _ := FromGitHub("pull_request", "del-2", []byte(pr("opened", false)), now)
	if !slices.Equal(n.Hints.CorrelationIDs, []string{reqID}) || n.Hints.PRNumber != 173 || n.Events[0].Metadata["prUrl"] != "https://gh/pr/173" {
		t.Fatalf("hints %+v events %+v", n.Hints, n.Events)
	}
	if n.Events[0].EventID != "github:del-2:GITOPS.PR.CREATED" {
		t.Fatalf("event id %s", n.Events[0].EventID)
	}
	n, _ = FromGitHub("push", "del-3", []byte(cases[len(cases)-1].body), now)
	if len(n.Hints.CorrelationIDs) != 1 {
		t.Fatal("push commit message must yield the correlation id")
	}
	if _, err := FromGitHub("issues", "d", []byte(`{}`), now); err == nil {
		t.Fatal("unsupported event must fail")
	}
	if _, err := FromGitHub("push", "d", []byte(`nope`), now); err == nil {
		t.Fatal("malformed payload must fail")
	}
}

func TestFromArgo(t *testing.T) {
	for event, want := range map[string]Type{"sync-running": GitOpsSyncStarted, "sync-succeeded": GitOpsSyncSucceeded,
		"sync-failed": GitOpsSyncFailed, "health-degraded": LifecycleDegraded} {
		body := `{"eventId":"uid-1-` + event + `","app":"claims","namespace":"argocd","event":"` + event + `","revision":"abc","timestamp":"2026-10-17T10:37:42Z"}`
		n, err := FromArgo([]byte(body), now)
		if err != nil {
			t.Fatal(err)
		}
		e := n.Events[0]
		if e.Type != want || e.Source != SourceArgoCD || n.Hints.ArgoApp != "argocd/claims" || e.Metadata["revision"] != "abc" {
			t.Fatalf("%s: %+v %+v", event, e, n.Hints)
		}
		if !e.Timestamp.Equal(time.Date(2026, 10, 17, 10, 37, 42, 0, time.UTC)) {
			t.Fatalf("timestamp %v", e.Timestamp)
		}
	}
	n, _ := FromArgo([]byte(`{"eventId":"x","app":"claims","event":"sync-running","timestamp":"bad"}`), now)
	if !n.Events[0].Timestamp.Equal(now) || n.Hints.ArgoApp != "claims" {
		t.Fatalf("fallbacks: %+v", n)
	}
	for _, bad := range []string{`{`, `{"app":"a","event":"sync-running"}`, `{"eventId":"x","event":"sync-running"}`, `{"eventId":"x","app":"a","event":"created"}`} {
		if _, err := FromArgo([]byte(bad), now); err == nil {
			t.Errorf("%s: expected error", bad)
		}
	}
}
