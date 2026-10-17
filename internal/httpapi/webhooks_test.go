package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/agnaldom/crosswatc/internal/auth"
)

type fakeInbox struct {
	rows map[string]string // source/delivery → event type
	err  error
}

func (f *fakeInbox) InsertInbox(_ context.Context, source, delivery, event string, _ json.RawMessage) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	k := source + "/" + delivery
	if _, dup := f.rows[k]; dup {
		return false, nil
	}
	f.rows[k] = event
	return true, nil
}

const ghSecret = "s3cr3t"

func sign(body string) string {
	m := hmac.New(sha256.New, []byte(ghSecret))
	m.Write([]byte(body))
	return "sha256=" + hex.EncodeToString(m.Sum(nil))
}

func webhookServer(inbox *fakeInbox, cfg WebhookConfig) http.Handler {
	return New(Deps{Store: &fakeStore{}, Auth: auth.Dev{}, Inbox: inbox, Webhooks: cfg})
}

func post(h http.Handler, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestGitHubWebhook(t *testing.T) {
	inbox := &fakeInbox{rows: map[string]string{}}
	h := webhookServer(inbox, WebhookConfig{GitHubSecret: ghSecret})
	body := `{"action":"opened","pull_request":{"number":7}}`
	hdr := func(sig, event, delivery string) map[string]string {
		return map[string]string{"X-Hub-Signature-256": sig, "X-GitHub-Event": event, "X-GitHub-Delivery": delivery}
	}

	cases := []struct {
		name    string
		body    string
		headers map[string]string
		code    int
		status  string
	}{
		{"valid", body, hdr(sign(body), "pull_request", "d-1"), http.StatusAccepted, "queued"},
		{"duplicate delivery", body, hdr(sign(body), "pull_request", "d-1"), http.StatusAccepted, "duplicate"},
		{"bad signature", body, hdr(sign(body+"x"), "pull_request", "d-2"), http.StatusUnauthorized, ""},
		{"missing signature", body, hdr("", "pull_request", "d-3"), http.StatusUnauthorized, ""},
		{"not hex", body, hdr("sha256=zz", "pull_request", "d-4"), http.StatusUnauthorized, ""},
		{"sha1 only", body, hdr("sha1=abc", "pull_request", "d-5"), http.StatusUnauthorized, ""},
		{"missing delivery", body, hdr(sign(body), "pull_request", ""), http.StatusBadRequest, ""},
		{"ping", `{}`, hdr(sign(`{}`), "ping", "d-6"), http.StatusOK, "pong"},
		{"unsupported event", `{}`, hdr(sign(`{}`), "issues", "d-7"), http.StatusAccepted, "ignored"},
		{"not json", `nope`, hdr(sign(`nope`), "push", "d-8"), http.StatusBadRequest, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := post(h, "/webhooks/github", tc.body, tc.headers)
			if rec.Code != tc.code {
				t.Fatalf("code %d body %s", rec.Code, rec.Body)
			}
			if tc.status != "" && !strings.Contains(rec.Body.String(), `"status":"`+tc.status+`"`) {
				t.Fatalf("body %s", rec.Body)
			}
		})
	}
	if len(inbox.rows) != 1 || inbox.rows["github/d-1"] != "pull_request" {
		t.Fatalf("only the valid delivery may be persisted: %v", inbox.rows)
	}
	inbox.err = errors.New("db down")
	if rec := post(h, "/webhooks/github", body, hdr(sign(body), "push", "d-9")); rec.Code != http.StatusInternalServerError {
		t.Fatalf("db failure must be 500 so GitHub retries, got %d", rec.Code)
	}
}

func TestArgoWebhook(t *testing.T) {
	inbox := &fakeInbox{rows: map[string]string{}}
	h := webhookServer(inbox, WebhookConfig{ArgoCDToken: "tok"})
	body := `{"eventId":"e-1","app":"claims","namespace":"argocd","event":"sync-succeeded"}`
	bearer := map[string]string{"Authorization": "Bearer tok"}

	if rec := post(h, "/webhooks/argocd", body, bearer); rec.Code != http.StatusAccepted || !strings.Contains(rec.Body.String(), "queued") {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if rec := post(h, "/webhooks/argocd", body, bearer); !strings.Contains(rec.Body.String(), "duplicate") {
		t.Fatalf("duplicate eventId: %s", rec.Body)
	}
	for name, hdr := range map[string]map[string]string{"wrong token": {"Authorization": "Bearer nope"}, "no token": {}} {
		if rec := post(h, "/webhooks/argocd", body, hdr); rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s: %d", name, rec.Code)
		}
	}
	for _, bad := range []string{`{"app":"a","event":"sync-failed"}`, `{"eventId":"x","app":"a","event":"deleted"}`, `[`} {
		if rec := post(h, "/webhooks/argocd", bad, bearer); rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d", bad, rec.Code)
		}
	}
	if len(inbox.rows) != 1 || inbox.rows["argocd/e-1"] != "sync-succeeded" {
		t.Fatalf("rows %v", inbox.rows)
	}
}

func TestWebhooksDisabledWithoutSecrets(t *testing.T) {
	h := webhookServer(&fakeInbox{rows: map[string]string{}}, WebhookConfig{})
	for _, p := range []string{"/webhooks/github", "/webhooks/argocd"} {
		if rec := post(h, p, `{}`, nil); rec.Code != http.StatusNotFound {
			t.Fatalf("%s: %d", p, rec.Code)
		}
	}
}
