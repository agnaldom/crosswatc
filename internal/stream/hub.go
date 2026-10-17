// Package stream fans realtime events out to SSE clients (spec §45). Database triggers append every change
// to stream_event and NOTIFY; each API replica runs one Hub that LISTENs, reads the new rows in id order and
// delivers them to its local subscribers. stream_event doubles as the Last-Event-ID replay buffer.
package stream

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/agnaldom/crosswatc/internal/store"
)

const (
	// ReplayWindow is how far back Last-Event-ID reconnections are served (§45: last 5 minutes).
	ReplayWindow = 5 * time.Minute
	// reorderSlack re-reads this many ids below the high-water mark: sequence ids are allocated before commit,
	// so a slower transaction can commit a lower id after a higher one was delivered.
	reorderSlack = 100
	batchSize    = 500
	bufferSize   = 256
)

// Source is the database side of the hub (satisfied by *store.Store).
type Source interface {
	StreamEventsAfter(ctx context.Context, afterID int64, since time.Time, limit int) ([]store.StreamEvent, error)
	LatestStreamID(ctx context.Context) (int64, error)
	Listen(ctx context.Context, channel string, onConnect func(), onNotify func(payload string)) error
}

// Filter selects the events a subscriber may receive.
type Filter func(store.StreamEvent) bool

// Subscription is one SSE client. C is closed when the client is too slow (it should reconnect with
// Last-Event-ID) or the hub stops.
type Subscription struct {
	C      <-chan store.StreamEvent
	ch     chan store.StreamEvent
	filter Filter
	once   sync.Once
}

func (s *Subscription) close() { s.once.Do(func() { close(s.ch) }) }

// Hub delivers stream events to local subscribers.
type Hub struct {
	src  Source
	log  *slog.Logger
	poll time.Duration // safety net in case a notification is lost

	mu   sync.RWMutex
	subs map[*Subscription]struct{}

	wake      chan struct{}
	high      int64
	delivered map[int64]struct{} // recent ids, to skip re-reads within reorderSlack
}

// NewHub builds a hub. poll is the catch-up interval used besides NOTIFY (default 10s).
func NewHub(src Source, log *slog.Logger, poll time.Duration) *Hub {
	if log == nil {
		log = slog.Default()
	}
	if poll <= 0 {
		poll = 10 * time.Second
	}
	return &Hub{src: src, log: log, poll: poll, subs: map[*Subscription]struct{}{}, wake: make(chan struct{}, 1),
		delivered: map[int64]struct{}{}}
}

// Subscribe registers a subscriber. Call Unsubscribe when the client goes away.
func (h *Hub) Subscribe(f Filter) *Subscription {
	ch := make(chan store.StreamEvent, bufferSize)
	s := &Subscription{C: ch, ch: ch, filter: f}
	h.mu.Lock()
	h.subs[s] = struct{}{}
	h.mu.Unlock()
	return s
}

// Unsubscribe removes and closes a subscriber.
func (h *Hub) Unsubscribe(s *Subscription) {
	h.mu.Lock()
	delete(h.subs, s)
	h.mu.Unlock()
	s.close()
}

// Replay returns buffered events after lastID that pass f (Last-Event-ID reconnection).
func (h *Hub) Replay(ctx context.Context, lastID int64, f Filter) ([]store.StreamEvent, error) {
	var out []store.StreamEvent
	since := time.Now().Add(-ReplayWindow)
	for {
		evs, err := h.src.StreamEventsAfter(ctx, lastID, since, batchSize)
		if err != nil {
			return nil, err
		}
		for _, e := range evs {
			if f == nil || f(e) {
				out = append(out, e)
			}
			lastID = e.ID
		}
		if len(evs) < batchSize {
			return out, nil
		}
	}
}

func (h *Hub) poke() {
	select {
	case h.wake <- struct{}{}:
	default:
	}
}

// Run listens for notifications and dispatches events until ctx ends, reconnecting with backoff.
func (h *Hub) Run(ctx context.Context) error {
	high, err := h.src.LatestStreamID(ctx)
	if err != nil {
		return err
	}
	h.high = high
	// rows already in the slack window predate this hub: never broadcast them
	old, err := h.src.StreamEventsAfter(ctx, max(high-reorderSlack, 0), time.Now().Add(-ReplayWindow), batchSize)
	if err != nil {
		return err
	}
	for _, e := range old {
		h.delivered[e.ID] = struct{}{}
	}
	go h.listen(ctx)
	t := time.NewTicker(h.poll)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			h.closeAll()
			return ctx.Err()
		case <-h.wake:
		case <-t.C:
		}
		if err := h.dispatch(ctx); err != nil && ctx.Err() == nil {
			h.log.Warn("stream dispatch failed", "err", err)
		}
	}
}

func (h *Hub) listen(ctx context.Context) {
	backoff := time.Second
	for {
		err := h.src.Listen(ctx, store.StreamChannel, func() { backoff = time.Second; h.poke() }, func(string) { h.poke() })
		if ctx.Err() != nil {
			return
		}
		h.log.Warn("stream LISTEN connection lost; reconnecting", "err", err, "in", backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, 30*time.Second)
	}
}

// dispatch reads new rows (including a slack below the high-water mark) and broadcasts unseen ones in id order.
func (h *Hub) dispatch(ctx context.Context) error {
	since := time.Now().Add(-ReplayWindow)
	for {
		evs, err := h.src.StreamEventsAfter(ctx, max(h.high-reorderSlack, 0), since, batchSize)
		if err != nil {
			return err
		}
		fresh := 0
		for _, e := range evs {
			if _, seen := h.delivered[e.ID]; seen {
				continue
			}
			h.delivered[e.ID] = struct{}{}
			h.high = max(h.high, e.ID)
			h.broadcast(e)
			fresh++
		}
		h.gc()
		if len(evs) < batchSize || fresh == 0 {
			return nil
		}
	}
}

func (h *Hub) gc() {
	cutoff := h.high - 2*reorderSlack
	for id := range h.delivered {
		if id < cutoff {
			delete(h.delivered, id)
		}
	}
}

func (h *Hub) broadcast(e store.StreamEvent) {
	h.mu.RLock()
	var slow []*Subscription
	for s := range h.subs {
		if s.filter != nil && !s.filter(e) {
			continue
		}
		select {
		case s.ch <- e:
		default:
			slow = append(slow, s)
		}
	}
	h.mu.RUnlock()
	for _, s := range slow {
		h.log.Warn("dropping slow stream subscriber; it will resume with Last-Event-ID")
		h.Unsubscribe(s)
	}
}

func (h *Hub) closeAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for s := range h.subs {
		delete(h.subs, s)
		s.close()
	}
}
