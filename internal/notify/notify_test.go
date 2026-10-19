package notify

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/agnaldom/crosswatc/internal/store"
)

type rec struct {
	channel string
	calls   [][]string
	msgs    []Message
	err     error
}

func (r *rec) Channel() string { return r.channel }
func (r *rec) Send(_ context.Context, to []string, m Message) error {
	r.calls, r.msgs = append(r.calls, to), append(r.msgs, m)
	return r.err
}

func notification(owner, kind string) store.Notification {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	return store.Notification{ID: 1, ResourceID: &id, Owner: owner, Kind: kind,
		Payload: []byte(`{"eventType":"INFRA.RECONCILE.FAILED","message":"NicInUse","resourceName":"vm-a","environment":"prd","kind":"VirtualMachine","prUrl":"https://gh/pr/1","correlationId":"crosswatc-req-X"}`)}
}

var cfg = Config{
	Default: &Route{Email: []string{"plataforma@corp.com"}, Kinds: []string{KindFailed}},
	Squads: map[string]Route{
		"payments": {Teams: []string{"https://teams/hook"}, Email: []string{"payments@corp.com", "plataforma@corp.com"}},
		"cards":    {Email: []string{"cards@corp.com"}, Kinds: []string{KindWaitingApproval}},
	},
}

func TestDispatcherRouting(t *testing.T) {
	teams, mail := &rec{channel: "teams"}, &rec{channel: "email"}
	d := &Dispatcher{Config: cfg, Teams: teams, Email: mail, PortalURL: "https://crosswatc.corp/"}
	if err := d.Handle(context.Background(), notification("payments", KindFailed)); err != nil {
		t.Fatal(err)
	}
	if len(teams.calls) != 1 || strings.Join(mail.calls[0], ",") != "payments@corp.com,plataforma@corp.com" {
		t.Fatalf("recipients must merge squad + default without duplicates: %v %v", teams.calls, mail.calls)
	}
	m := mail.msgs[0]
	if m.Link != "https://crosswatc.corp/infrastructure/11111111-1111-1111-1111-111111111111" || m.Title() != "Falha — vm-a (prd)" || m.Owner != "payments" {
		t.Fatalf("%+v", m)
	}

	mail.calls = nil
	if err := d.Handle(context.Background(), notification("cards", KindOperational)); !errors.Is(err, store.ErrSkip) {
		t.Fatalf("filtered kind with no default route must skip, got %v", err)
	}
	if err := d.Handle(context.Background(), notification("unknown", KindFailed)); err != nil || strings.Join(mail.calls[0], ",") != "plataforma@corp.com" {
		t.Fatalf("default route: %v %v", err, mail.calls)
	}
}

func TestDispatcherFailures(t *testing.T) {
	teams, mail := &rec{channel: "teams", err: errors.New("down")}, &rec{channel: "email"}
	d := &Dispatcher{Config: cfg, Teams: teams, Email: mail}
	if err := d.Handle(context.Background(), notification("payments", KindFailed)); err != nil {
		t.Fatalf("partial success must not retry: %v", err)
	}
	mail.err = errors.New("smtp down")
	if err := d.Handle(context.Background(), notification("payments", KindFailed)); err == nil || errors.Is(err, store.ErrSkip) {
		t.Fatalf("all channels failing must retry: %v", err)
	}
	bad := notification("payments", KindFailed)
	bad.Payload = []byte(`{`)
	if err := d.Handle(context.Background(), bad); !errors.Is(err, store.ErrPermanent) {
		t.Fatalf("bad payload is permanent: %v", err)
	}
}

func TestTeamsCard(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()
	m := Message{Kind: KindDriftDetected, ResourceName: "vm-a", Environment: "dev", Text: "drift", Link: "https://x", PRURL: "https://pr"}
	if err := (Teams{HTTP: srv.Client()}).Send(context.Background(), []string{srv.URL}, m); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(got)
	for _, want := range []string{`"contentType":"application/vnd.microsoft.card.adaptive"`, `"Drift de configuração detectado — vm-a (dev)"`, `"Action.OpenUrl"`, `"color":"Warning"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("card missing %s", want)
		}
	}
	fail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(400) }))
	defer fail.Close()
	if err := (Teams{}).Send(context.Background(), []string{fail.URL}, m); err == nil {
		t.Fatal("HTTP 400 must error")
	}
}

func TestEmail(t *testing.T) {
	var sentTo []string
	var body string
	e := Email{Addr: "smtp.corp:587", From: "crosswatc@corp.com", Username: "u", Password: "p",
		send: func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
			if addr != "smtp.corp:587" || a == nil || from != "crosswatc@corp.com" {
				t.Errorf("args %s %v %s", addr, a, from)
			}
			sentTo, body = to, string(msg)
			return nil
		}}
	m := Message{Kind: KindWaitingApproval, ResourceName: "vm-a", Environment: "prd\r\nBcc: evil@x", Text: "Aguardando", PRURL: "https://pr"}
	if err := e.Send(context.Background(), []string{"a@corp.com"}, m); err != nil {
		t.Fatal(err)
	}
	if sentTo[0] != "a@corp.com" || !strings.Contains(body, "Subject: [Crosswatc] Aguardando aprovação — vm-a (prdBcc: evil@x)") {
		t.Fatalf("header injection must be neutralised:\n%s", body)
	}
	headers, _, _ := strings.Cut(body, "\r\n\r\n")
	if strings.Contains(headers, "\r\nBcc:") {
		t.Fatal("injected header")
	}
}

func TestLoadConfig(t *testing.T) {
	p := filepath.Join(t.TempDir(), "n.yaml")
	_ = os.WriteFile(p, []byte("default: {email: [p@corp.com]}\nsquads:\n  payments: {teams: [\"https://h\"], kinds: [FAILED]}\n"), 0o600)
	c, err := LoadConfig(p)
	if err != nil || c.Squads["payments"].Teams[0] != "https://h" || c.Default.Email[0] != "p@corp.com" {
		t.Fatalf("%+v %v", c, err)
	}
	if _, err := LoadConfig(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing file")
	}
}
