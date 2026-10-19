package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ErrSkip marks a notification that has no recipient (status skipped, not retried).
var ErrSkip = errors.New("store: notification skipped")

// Notification is an outbox row (migration 00004).
type Notification struct {
	ID         int64
	EventID    uuid.UUID
	ResourceID *uuid.UUID
	Owner      string
	Kind       string
	Payload    json.RawMessage
	Attempts   int
	CreatedAt  time.Time
}

// ProcessNotifications locks due notifications (SKIP LOCKED, safe across replicas) and records the outcome of fn:
// nil → sent; ErrSkip → skipped; ErrPermanent or maxAttempts reached → failed; otherwise retried with
// quadratic backoff (30s, 2m, 4.5m…).
func (s *Store) ProcessNotifications(ctx context.Context, limit, maxAttempts int, fn func(Notification) error) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
SELECT id, event_id, resource_id, coalesce(owner,''), kind, payload, attempts, created_at FROM notification
WHERE status = 'pending' AND next_attempt_at <= now() ORDER BY id LIMIT $1 FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return 0, err
	}
	var items []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.EventID, &n.ResourceID, &n.Owner, &n.Kind, &n.Payload, &n.Attempts, &n.CreatedAt); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, n)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, n := range items {
		ferr := fn(n)
		switch {
		case ferr == nil:
			_, err = tx.Exec(ctx, `UPDATE notification SET status='sent', sent_at=now(), attempts=attempts+1, last_error=NULL WHERE id=$1`, n.ID)
		case errors.Is(ferr, ErrSkip):
			_, err = tx.Exec(ctx, `UPDATE notification SET status='skipped', last_error=$2 WHERE id=$1`, n.ID, ferr.Error())
		default:
			_, err = tx.Exec(ctx, `
UPDATE notification SET attempts = attempts + 1, last_error = $2,
  status = CASE WHEN attempts + 1 >= $3 OR $4 THEN 'failed' ELSE 'pending' END,
  next_attempt_at = now() + make_interval(secs => 30 * (attempts + 1) * (attempts + 1))
WHERE id = $1`, n.ID, ferr.Error(), maxAttempts, errors.Is(ferr, ErrPermanent))
		}
		if err != nil {
			return 0, fmt.Errorf("store: notification %d: %w", n.ID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(items), nil
}
