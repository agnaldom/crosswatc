package httpapi

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/agnaldom/crosswatc/internal/auth"
	"github.com/agnaldom/crosswatc/internal/store"
	"github.com/agnaldom/crosswatc/internal/stream"
)

// fakeStreamSource is an in-memory stream_event table behind a real stream.Hub.
type fakeStreamSource struct {
	mu     sync.Mutex
	rows   []store.StreamEvent
	notify chan string
}

func (f *fakeStreamSource) add(id int64, typ string, res *uuid.UUID, owner string) {
	f.mu.Lock()
	f.rows = append(f.rows, store.StreamEvent{ID: id, Type: typ, ResourceID: res, Owner: owner,
		Data: json.RawMessage(`{"resourceId": "x",` + "\n" + ` "n": 1}`), CreatedAt: time.Now()})
	f.mu.Unlock()
}

func (f *fakeStreamSource) StreamEventsAfter(_ context.Context, after int64, since time.Time, limit int) ([]store.StreamEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []store.StreamEvent
	for _, r := range f.rows {
		if r.ID > after && !r.CreatedAt.Before(since) {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeStreamSource) LatestStreamID(context.Context) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var hi int64
	for _, r := range f.rows {
		hi = max(hi, r.ID)
	}
	return hi, nil
}

func (f *fakeStreamSource) Listen(ctx context.Context, _ string, onConnect func(), onNotify func(string)) error {
	onConnect()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case p := <-f.notify:
			onNotify(p)
		}
	}
}

type sseMsg struct{ id, event, data, comment string }

// sseClient reads SSE messages from a live connection.
type sseClient struct {
	resp *http.Response
	msgs chan sseMsg
}

func connect(t *testing.T, url string, hdr map[string]string) *sseClient {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	c := &sseClient{resp: resp, msgs: make(chan sseMsg, 32)}
	if resp.StatusCode != http.StatusOK {
		return c
	}
	go func() {
		defer close(c.msgs)
		sc := bufio.NewScanner(resp.Body)
		var m sseMsg
		for sc.Scan() {
			line := sc.Text()
			switch {
			case line == "":
				if m != (sseMsg{}) {
					c.msgs <- m
				}
				m = sseMsg{}
			case strings.HasPrefix(line, ":"):
				m.comment = strings.TrimSpace(line[1:])
			default:
				k, v, _ := strings.Cut(line, ": ")
				switch k {
				case "id":
					m.id = v
				case "event":
					m.event = v
				case "data":
					m.data = v
				}
			}
		}
	}()
	return c
}

// next returns the next message with an id (or the next comment when wantComment).
func (c *sseClient) next(t *testing.T, wantComment bool) sseMsg {
	t.Helper()
	timeout := time.After(3 * time.Second)
	for {
		select {
		case m, ok := <-c.msgs:
			if !ok {
				t.Fatal("stream closed")
			}
			if (wantComment && m.comment != "") || (!wantComment && m.id != "") {
				return m
			}
		case <-timeout:
			t.Fatal("timeout waiting for SSE message")
		}
	}
}

func (c *sseClient) none(t *testing.T) {
	t.Helper()
	deadline := time.After(150 * time.Millisecond)
	for {
		select {
		case m := <-c.msgs:
			if m.id != "" {
				t.Fatalf("unexpected event %+v", m)
			}
		case <-deadline:
			return
		}
	}
}

func streamSetup(t *testing.T, heartbeat time.Duration) (*fakeStreamSource, *httptest.Server) {
	t.Helper()
	fs, _, _ := setup()
	src := &fakeStreamSource{notify: make(chan string, 16)}
	src.add(1, "resource.updated", &payID, "payments") // buffered before the clients connect
	src.add(2, "resource.updated", &cardID, "cards")
	hub := stream.NewHub(src, nil, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = hub.Run(ctx) }()
	h := New(Deps{Store: fs, Auth: auth.Dev{Mapping: auth.GroupMapping{SquadPrefix: "squad-", SREGroup: "sre"}},
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Stream: hub, StreamHeartbeat: heartbeat})
	srv := httptest.NewServer(h)
	t.Cleanup(func() { cancel(); srv.Close() })
	time.Sleep(50 * time.Millisecond) // hub initialised
	return src, srv
}

func TestStreamAuthZAndResourceFilter(t *testing.T) {
	src, srv := streamSetup(t, time.Hour)
	sre := map[string]string{"X-Dev-User": "ops", "X-Dev-SRE": "true"}

	pay := connect(t, srv.URL+"/api/v1/stream", payments)
	if ct := pay.resp.Header.Get("Content-Type"); pay.resp.StatusCode != 200 || ct != "text/event-stream" {
		t.Fatalf("%d %s", pay.resp.StatusCode, ct)
	}
	all := connect(t, srv.URL+"/api/v1/stream", sre)
	onlyPay := connect(t, srv.URL+"/api/v1/stream?resourceId="+payID.String(), sre)
	time.Sleep(50 * time.Millisecond)

	src.add(3, "event.created", &cardID, "cards")
	src.add(4, "revision.updated", &payID, "payments")
	src.add(5, "event.created", nil, "") // unknown resource: SRE only
	src.notify <- "x"

	if m := pay.next(t, false); m.id != "4" || m.event != "revision.updated" || m.data != `{"resourceId":"x","n":1}` {
		t.Fatalf("payments got %+v", m)
	}
	pay.none(t)
	for _, want := range []string{"3", "4", "5"} {
		if m := all.next(t, false); m.id != want {
			t.Fatalf("sre got %s want %s", m.id, want)
		}
	}
	if m := onlyPay.next(t, false); m.id != "4" {
		t.Fatalf("resource filter got %+v", m)
	}
	onlyPay.none(t)
}

func TestStreamRejects(t *testing.T) {
	_, srv := streamSetup(t, time.Hour)
	for name, tc := range map[string]struct {
		path string
		hdr  map[string]string
		code int
	}{
		"unauthenticated":     {"/api/v1/stream", nil, http.StatusUnauthorized},
		"foreign resource":    {"/api/v1/stream?resourceId=" + cardID.String(), payments, http.StatusNotFound},
		"unknown resource":    {"/api/v1/stream?resourceId=" + uuid.NewString(), payments, http.StatusNotFound},
		"malformed resource":  {"/api/v1/stream?resourceId=nope", payments, http.StatusNotFound},
		"bad last event id":   {"/api/v1/stream", map[string]string{"X-Dev-User": "a", "X-Dev-Squads": "payments", "Last-Event-ID": "x"}, http.StatusBadRequest},
		"negative last event": {"/api/v1/stream?lastEventId=-1", payments, http.StatusBadRequest},
	} {
		t.Run(name, func(t *testing.T) {
			c := connect(t, srv.URL+tc.path, tc.hdr)
			if c.resp.StatusCode != tc.code {
				t.Fatalf("got %d", c.resp.StatusCode)
			}
		})
	}
	fs, _, _ := setup()
	h := New(Deps{Store: fs, Auth: auth.Dev{}})
	if rec := do(h, "/api/v1/stream", map[string]string{"X-Dev-User": "a"}); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("no hub: %d", rec.Code)
	}
}

func TestStreamLastEventIDReplayAndHeartbeat(t *testing.T) {
	src, srv := streamSetup(t, 50*time.Millisecond)
	hdr := map[string]string{"X-Dev-User": "agnaldo", "X-Dev-Squads": "payments,cards", "Last-Event-ID": "0"}
	c := connect(t, srv.URL+"/api/v1/stream", hdr)
	// buffered events after Last-Event-ID are replayed first, filtered by AuthZ, in id order
	if a, b := c.next(t, false), c.next(t, false); a.id != "1" || b.id != "2" {
		t.Fatalf("replay %s %s", a.id, b.id)
	}
	if m := c.next(t, true); m.comment != "ping" {
		t.Fatalf("heartbeat %+v", m)
	}
	src.add(3, "topology.updated", &payID, "payments")
	src.notify <- "x"
	if m := c.next(t, false); m.id != "3" || m.event != "topology.updated" {
		t.Fatalf("live %+v", m)
	}

	resume := connect(t, srv.URL+"/api/v1/stream?lastEventId=2", payments)
	if m := resume.next(t, false); m.id != "3" {
		t.Fatalf("resume %+v", m)
	}
	resume.none(t) // replayed events are not delivered twice
}
