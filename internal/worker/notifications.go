package worker

import (
	"context"
	"time"

	"github.com/agnaldom/crosswatc/internal/store"
)

// NotificationStore is the outbox (satisfied by *store.Store).
type NotificationStore interface {
	ProcessNotifications(ctx context.Context, limit, maxAttempts int, fn func(store.Notification) ([]string, error)) (int, error)
}

// Notifier delivers one notification (satisfied by *notify.Dispatcher).
type Notifier interface {
	Handle(ctx context.Context, n store.Notification) ([]string, error)
}

const (
	notifyBatch       = 50
	notifyMaxAttempts = 6
)

// consumeNotifications drains the outbox (spec §60). SKIP LOCKED makes concurrent replicas safe.
func (r *Runner) consumeNotifications(ctx context.Context) {
	interval := r.NotifyInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		for {
			n, err := r.Notifications.ProcessNotifications(ctx, notifyBatch, notifyMaxAttempts, func(it store.Notification) ([]string, error) {
				return r.Notifier.Handle(ctx, it)
			})
			if err != nil && ctx.Err() == nil {
				r.log().Warn("notification delivery failed", "err", err)
			}
			if err != nil || n < notifyBatch {
				break
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
