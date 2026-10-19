package worker

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/agnaldom/crosswatc/internal/store"
)

type fakeOutbox struct {
	mu    sync.Mutex
	items []store.Notification
	done  []int64
}

func (f *fakeOutbox) ProcessNotifications(_ context.Context, limit, _ int, fn func(store.Notification) error) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for len(f.items) > 0 && n < limit {
		it := f.items[0]
		f.items = f.items[1:]
		if fn(it) == nil {
			f.done = append(f.done, it.ID)
		}
		n++
	}
	return n, nil
}

type countNotifier struct{ n int }

func (c *countNotifier) Handle(context.Context, store.Notification) error { c.n++; return nil }

func TestConsumeNotifications(t *testing.T) {
	ob := &fakeOutbox{}
	for i := range 120 {
		ob.items = append(ob.items, store.Notification{ID: int64(i)})
	}
	nt := &countNotifier{}
	r := &Runner{Notifications: ob, Notifier: nt, NotifyInterval: 10 * time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	r.consumeNotifications(ctx)
	ob.mu.Lock()
	defer ob.mu.Unlock()
	if len(ob.done) != 120 || nt.n != 120 {
		t.Fatalf("delivered %d/%d", len(ob.done), nt.n)
	}
}
