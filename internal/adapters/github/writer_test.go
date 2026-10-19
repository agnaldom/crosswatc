package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriter(t *testing.T) {
	var got []string
	mux := http.NewServeMux()
	write := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}
	mux.HandleFunc("GET /repos/org/argocd/contents/claims/dev/vm.yaml", func(w http.ResponseWriter, r *http.Request) {
		got = append(got, "get@"+r.URL.Query().Get("ref"))
		write(w, map[string]any{"type": "file", "encoding": "base64", "sha": "blob1",
			"content": base64.StdEncoding.EncodeToString([]byte("kind: VirtualMachine\n"))})
	})
	mux.HandleFunc("GET /repos/org/argocd/contents/missing.yaml", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("GET /repos/org/argocd/git/ref/heads/main", func(w http.ResponseWriter, _ *http.Request) {
		write(w, map[string]any{"ref": "refs/heads/main", "object": map[string]any{"sha": "base1"}})
	})
	mux.HandleFunc("POST /repos/org/argocd/git/refs", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = append(got, "ref:"+string(b))
		write(w, map[string]any{"ref": "refs/heads/x"})
	})
	mux.HandleFunc("PUT /repos/org/argocd/contents/claims/dev/vm.yaml", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		got = append(got, "put:"+body["branch"].(string)+":"+body["sha"].(string))
		write(w, map[string]any{"commit": map[string]any{"sha": "commit1"}})
	})
	mux.HandleFunc("DELETE /repos/org/argocd/contents/claims/dev/vm.yaml", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		got = append(got, "del:"+body["branch"].(string)+":"+body["sha"].(string))
		write(w, map[string]any{"commit": map[string]any{"sha": "commit-del"}})
	})
	mux.HandleFunc("POST /repos/org/argocd/pulls", func(w http.ResponseWriter, _ *http.Request) {
		write(w, map[string]any{"number": 173, "html_url": "https://gh/pr/173"})
	})
	mux.HandleFunc("POST /repos/org/argocd/issues/173/comments", func(w http.ResponseWriter, _ *http.Request) {
		got = append(got, "comment")
		write(w, map[string]any{"id": 1})
	})
	mux.HandleFunc("PATCH /repos/org/argocd/pulls/173", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		got = append(got, "state:"+body["state"].(string))
		write(w, map[string]any{"number": 173})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	a, _ := NewWithHTTPClient(srv.Client(), srv.URL)
	ctx := context.Background()

	content, sha, err := a.GetFile(ctx, "org/argocd", "claims/dev/vm.yaml", "main")
	if err != nil || string(content) != "kind: VirtualMachine\n" || sha != "blob1" {
		t.Fatalf("%q %q %v", content, sha, err)
	}
	if _, _, err := a.GetFile(ctx, "org/argocd", "missing.yaml", "main"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
	head, err := a.BranchHead(ctx, "org/argocd", "main")
	if err != nil || head != "base1" {
		t.Fatal(head, err)
	}
	if err := a.CreateBranch(ctx, "org/argocd", "update/x", head); err != nil {
		t.Fatal(err)
	}
	c, err := a.CommitFile(ctx, "org/argocd", "update/x", "claims/dev/vm.yaml", []byte("y"), "blob1", "msg")
	if err != nil || c != "commit1" {
		t.Fatal(c, err)
	}
	if c, err := a.DeleteFile(ctx, "org/argocd", "update/x", "claims/dev/vm.yaml", "blob1", "rm"); err != nil || c != "commit-del" {
		t.Fatal(c, err)
	}
	n, url, err := a.OpenPullRequest(ctx, "org/argocd", "update/x", "main", "t", "b")
	if err != nil || n != 173 || url != "https://gh/pr/173" {
		t.Fatal(n, url, err)
	}
	if err := a.ClosePullRequest(ctx, "org/argocd", 173, "bye"); err != nil {
		t.Fatal(err)
	}
	want := []string{"get@main", `ref:{"ref":"refs/heads/update/x","sha":"base1"}`, "put:update/x:blob1", "del:update/x:blob1", "comment", "state:closed"}
	for i, w := range want {
		if i >= len(got) || strings.TrimSpace(got[i]) != w {
			t.Fatalf("calls %v, want %v", got, want)
		}
	}
	if _, _, err := a.GetFile(ctx, "bad", "p", "main"); err == nil {
		t.Fatal("invalid repository")
	}
}
