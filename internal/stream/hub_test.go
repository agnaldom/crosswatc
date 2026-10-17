package stream

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/agnaldom/crosswatc/internal/store"
)

// memSource is an in-memory stream_event table with a fake LISTEN channel.
type memSource struct {
	mu       sync.Mutex
	rows     []store.StreamEvent
	notify   chan string
	listens  int
	failNext bool
}

func newMemSource() *memSource { return &memSource{notify: make(chan string, 64)} }

func (m *memSource) add(id int64, typ, owner string) {
	m.mu.Lock()
	m.rows = append(m.rows, store.StreamEvent{ID: id, Type: typ, Owner: owner, Data: json.RawMessage(`{}`), CreatedAt: time.Now()})
	m.mu.Unlock()
}

func (m *memSource) publish(id int64, typ, owner string) {
	m.add(id, typ, owner)
	m.notify <- "x"
}

func (m *memSource) StreamEventsAfter(_ context.Context, after int64, since time.Time, limit int) ([]store.StreamEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []store.StreamEvent
	for _, r := range m.rows {
		if r.ID > after && !r.CreatedAt.Before(since) {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID }) // ORDER BY id
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *memSource) LatestStreamID(context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var hi int64
	for _, r := range m.rows {
		hi = max(hi, r.ID)
	}
	return hi, nil
}

func (m *memSource) Listen(ctx context.Context, _ string, onConnect func(), onNotify func(string)) error {
	m.mu.Lock()
	m.listens++
	fail := m.failNext
	m.failNext = false
	m.mu.Unlock()
	if fail {
		return errors.New("connection refused")
	}
	onConnect()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case p := <-m.notify:
			onNotify(p)
		}
	}
}

func recv(t *testing.T, s *Subscription) store.StreamEvent {
	t.Helper()
	select {
	case e, ok := <-s.C:
		if !ok {
			t.Fatal("subscription closed")
		}
		return e
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
		return store.StreamEvent{}
	}
}

func TestHubFanOutFilterAndOrder(t *testing.T) {
	src := newMemSource()
	src.add(1, "resource.updated", "payments") // before the hub started: never broadcast
	h := NewHub(src, nil, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- h.Run(ctx) }()

	payments := h.Subscribe(func(e store.StreamEvent) bool { return e.Owner == "payments" })
	all := h.Subscribe(nil)
	time.Sleep(50 * time.Millisecond)

	src.publish(2, "revision.updated", "payments")
	src.publish(3, "event.created", "data")
	if e := recv(t, payments); e.ID != 2 {
		t.Fatalf("payments got %d", e.ID)
	}
	if a, b := recv(t, all), recv(t, all); a.ID != 2 || b.ID != 3 {
		t.Fatalf("all got %d %d", a.ID, b.ID)
	}

	// a lower id committed late (sequence allocated before commit) is still delivered, once
	src.publish(5, "event.created", "payments")
	src.publish(4, "event.created", "payments")
	got := map[int64]bool{recv(t, payments).ID: true, recv(t, payments).ID: true}
	if !got[4] || !got[5] {
		t.Fatalf("late commit lost: %v", got)
	}
	src.notify <- "again"
	select {
	case e := <-payments.C:
		t.Fatalf("duplicate delivery %d", e.ID)
	case <-time.After(100 * time.Millisecond):
	}

	h.Unsubscribe(payments)
	if _, ok := <-payments.C; ok {
		t.Fatal("unsubscribed channel must be closed")
	}
	cancel()
	<-done
	for range all.C { // drain anything buffered; the channel must end closed
	}
}

func TestHubDropsSlowSubscriber(t *testing.T) {
	src := newMemSource()
	h := NewHub(src, nil, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = h.Run(ctx) }()
	slow := h.Subscribe(nil)
	time.Sleep(50 * time.Millisecond)
	for i := int64(1); i <= bufferSize+10; i++ {
		src.add(i, "event.created", "x")
	}
	src.notify <- "x"
	time.Sleep(300 * time.Millisecond) // the client is not reading while the hub dispatches
	deadline := time.After(3 * time.Second)
	n := 0
	for {
		select {
		case _, ok := <-slow.C:
			if !ok {
				if n > bufferSize {
					t.Fatalf("received %d > buffer", n)
				}
				return // dropped: client will reconnect with Last-Event-ID
			}
			n++
		case <-deadline:
			t.Fatal("slow subscriber not dropped")
		}
	}
}

func TestHubReconnectsAndReplays(t *testing.T) {
	src := newMemSource()
	src.failNext = true
	h := NewHub(src, nil, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = h.Run(ctx) }()
	sub := h.Subscribe(nil)
	time.Sleep(50 * time.Millisecond)
	src.add(7, "event.created", "x") // written while LISTEN was down
	// listen fails once, reconnects after backoff and catches up on connect
	if e := recv(t, sub); e.ID != 7 {
		t.Fatalf("got %d", e.ID)
	}
	src.mu.Lock()
	listens := src.listens
	src.mu.Unlock()
	if listens < 2 {
		t.Fatalf("listens %d", listens)
	}

	evs, err := h.Replay(ctx, 0, func(e store.StreamEvent) bool { return e.ID != 7 })
	if err != nil || len(evs) != 0 {
		t.Fatalf("filtered replay: %v %v", evs, err)
	}
	evs, _ = h.Replay(ctx, 6, nil)
	if len(evs) != 1 || evs[0].ID != 7 {
		t.Fatalf("replay %v", evs)
	}
}
