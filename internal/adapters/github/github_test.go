package github

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agnaldom/crosswatc/internal/lifecycle"
)

type fakeGH struct {
	pr        map[string]any
	searchHit int
	runs      []map[string]any
	statuses  []map[string]any
	reviews   []map[string]any
	commitPRs []map[string]any
	calls     []string
}

func (f *fakeGH) server(t *testing.T) *Adapter {
	t.Helper()
	mux := http.NewServeMux()
	write := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}
	mux.HandleFunc("GET /search/issues", func(w http.ResponseWriter, r *http.Request) {
		f.calls = append(f.calls, "search:"+r.URL.Query().Get("q"))
		var items []map[string]any
		if f.searchHit != 0 {
			items = append(items, map[string]any{"number": f.searchHit})
		}
		write(w, map[string]any{"total_count": len(items), "items": items})
	})
	mux.HandleFunc("GET /repos/org/argocd/commits", func(w http.ResponseWriter, r *http.Request) {
		f.calls = append(f.calls, "commits:"+r.URL.Query().Get("path"))
		write(w, []map[string]any{{"sha": "c1"}})
	})
	mux.HandleFunc("GET /repos/org/argocd/commits/c1/pulls", func(w http.ResponseWriter, _ *http.Request) {
		write(w, f.commitPRs)
	})
	mux.HandleFunc("GET /repos/org/argocd/pulls/{n}", func(w http.ResponseWriter, r *http.Request) {
		f.calls = append(f.calls, "pr:"+r.PathValue("n"))
		write(w, f.pr)
	})
	mux.HandleFunc("GET /repos/org/argocd/commits/head1/check-runs", func(w http.ResponseWriter, _ *http.Request) {
		write(w, map[string]any{"total_count": len(f.runs), "check_runs": f.runs})
	})
	mux.HandleFunc("GET /repos/org/argocd/commits/head1/status", func(w http.ResponseWriter, _ *http.Request) {
		write(w, map[string]any{"statuses": f.statuses})
	})
	mux.HandleFunc("GET /repos/org/argocd/pulls/{n}/reviews", func(w http.ResponseWriter, _ *http.Request) {
		write(w, f.reviews)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	a, err := NewWithHTTPClient(srv.Client(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func openPR() map[string]any {
	return map[string]any{"number": 173, "state": "open", "merged": false, "html_url": "https://github.com/org/argocd/pull/173",
		"head": map[string]any{"sha": "head1"}, "merge_commit_sha": "tmp"}
}

func review(user, state string) map[string]any {
	return map[string]any{"user": map[string]any{"login": user}, "state": state}
}

func TestPullRequestByCorrelation(t *testing.T) {
	f := &fakeGH{pr: openPR(), searchHit: 173,
		runs: []map[string]any{
			{"status": "completed", "conclusion": "success"},
			{"status": "completed", "conclusion": "skipped"},
		},
		statuses: []map[string]any{{"state": "success"}},
		reviews:  []map[string]any{review("ana", "CHANGES_REQUESTED"), review("ana", "APPROVED"), review("bob", "COMMENTED")},
	}
	a := f.server(t)
	s, err := a.PullRequest(context.Background(), Lookup{Repository: "org/argocd", CorrelationID: "crosswatc-req-X", Path: "p.yaml"})
	if err != nil {
		t.Fatal(err)
	}
	want := lifecycle.PRSignal{Found: true, Number: 173, URL: "https://github.com/org/argocd/pull/173", Open: true,
		Checks: lifecycle.ChecksSuccess, Approved: true}
	if s != want {
		t.Fatalf("got %+v\nwant %+v", s, want)
	}
	if !strings.Contains(f.calls[0], `is:pr "crosswatc-req-X"`) {
		t.Fatalf("search query: %v", f.calls)
	}
	for _, c := range f.calls {
		if strings.HasPrefix(c, "commits:") {
			t.Fatal("path fallback must not run when correlation search hits")
		}
	}
}

func TestPullRequestByPathMerged(t *testing.T) {
	pr := openPR()
	pr["state"], pr["merged"], pr["merge_commit_sha"] = "closed", true, "m1"
	f := &fakeGH{pr: pr, commitPRs: []map[string]any{{"number": 9}, {"number": 173, "merged_at": "2026-10-17T10:00:00Z"}},
		runs: []map[string]any{{"status": "in_progress"}}}
	a := f.server(t)
	s, err := a.PullRequest(context.Background(), Lookup{Repository: "org/argocd", CorrelationID: "crosswatc-req-Y", Path: "claims/dev/pg.yaml"})
	if err != nil {
		t.Fatal(err)
	}
	if !s.Merged || s.MergeCommitSHA != "m1" || s.Checks != lifecycle.ChecksPending || s.Approved {
		t.Fatalf("got %+v", s)
	}
	if f.calls[len(f.calls)-1] != "pr:173" {
		t.Fatalf("must prefer merged PR: %v", f.calls)
	}
}

func TestPullRequestKnownNumberAndFailures(t *testing.T) {
	f := &fakeGH{pr: openPR(),
		runs:     []map[string]any{{"status": "completed", "conclusion": "success"}},
		statuses: []map[string]any{{"state": "error"}},
		reviews:  []map[string]any{review("ana", "APPROVED"), review("bob", "CHANGES_REQUESTED")},
	}
	a := f.server(t)
	s, err := a.PullRequest(context.Background(), Lookup{Repository: "org/argocd", PRNumber: 173})
	if err != nil {
		t.Fatal(err)
	}
	if s.Checks != lifecycle.ChecksFailure || s.Approved || !s.ChangesRequested || s.MergeCommitSHA != "" {
		t.Fatalf("got %+v", s)
	}
	if strings.HasPrefix(f.calls[0], "search:") {
		t.Fatal("known PR number must skip search")
	}
}

func TestPullRequestNotFound(t *testing.T) {
	a := (&fakeGH{}).server(t)
	s, err := a.PullRequest(context.Background(), Lookup{Repository: "org/argocd", CorrelationID: "crosswatc-req-Z"})
	if err != nil || s.Found {
		t.Fatalf("got %+v %v", s, err)
	}
	if _, err := a.PullRequest(context.Background(), Lookup{Repository: "bad"}); err == nil {
		t.Fatal("invalid repository must error")
	}
}

func TestAggregate(t *testing.T) {
	if aggregate(nil) != lifecycle.ChecksNone ||
		aggregate([]lifecycle.CheckState{lifecycle.ChecksSuccess, lifecycle.ChecksPending}) != lifecycle.ChecksPending ||
		aggregate([]lifecycle.CheckState{lifecycle.ChecksPending, lifecycle.ChecksFailure}) != lifecycle.ChecksFailure {
		t.Fatal("aggregate")
	}
}

func TestNewRequiresAppConfig(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("expected error")
	}
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	path := filepath.Join(t.TempDir(), "key.pem")
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(Config{AppID: 1, InstallationID: 2, PrivateKeyPath: path, APIURL: "https://ghe.example.com/api/v3"}); err != nil {
		t.Fatal(err)
	}
}
