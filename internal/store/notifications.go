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

// NotificationLease is how long a claimed notification stays reserved for one worker.
const NotificationLease = 2 * time.Minute

// Notification is an outbox row (migrations 00004/00005).
type Notification struct {
	ID          int64
	EventID     uuid.UUID
	ResourceID  *uuid.UUID
	Owner       string
	Kind        string
	Payload     json.RawMessage
	Attempts    int
	SentTargets []string // targets that already received it ("teams:<url>", "email:<addr>")
	CreatedAt   time.Time
}

// ClaimNotifications leases due rows in one short statement (no transaction held during delivery).
// Rows of a worker that died mid-delivery become due again when the lease expires.
func (s *Store) ClaimNotifications(ctx context.Context, limit int, lease time.Duration) ([]Notification, error) {
	rows, err := s.pool.Query(ctx, `
UPDATE notification SET status = 'sending', next_attempt_at = now() + make_interval(secs => $2)
WHERE id IN (
  SELECT id FROM notification WHERE status IN ('pending','sending') AND next_attempt_at <= now()
  ORDER BY id LIMIT $1 FOR UPDATE SKIP LOCKED)
RETURNING id, event_id, resource_id, coalesce(owner,''), kind, payload, attempts, sent_targets, created_at`,
		limit, lease.Seconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.EventID, &n.ResourceID, &n.Owner, &n.Kind, &n.Payload, &n.Attempts, &n.SentTargets, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// CompleteNotification records delivered targets and the outcome:
// nil → sent; ErrSkip → skipped; ErrPermanent or maxAttempts reached → failed;
// otherwise pending again with quadratic backoff (30s, 2m, 4.5m…).
func (s *Store) CompleteNotification(ctx context.Context, id int64, delivered []string, outcome error, maxAttempts int) error {
	if delivered == nil {
		delivered = []string{}
	}
	var err error
	switch {
	case outcome == nil:
		_, err = s.pool.Exec(ctx, `
UPDATE notification SET status='sent', sent_at=now(), attempts=attempts+1, last_error=NULL,
  sent_targets = ARRAY(SELECT DISTINCT unnest(sent_targets || $2::text[]))
WHERE id=$1`, id, delivered)
	case errors.Is(outcome, ErrSkip):
		_, err = s.pool.Exec(ctx, `UPDATE notification SET status='skipped', last_error=$2 WHERE id=$1`, id, outcome.Error())
	default:
		_, err = s.pool.Exec(ctx, `
UPDATE notification SET attempts = attempts + 1, last_error = $2,
  sent_targets = ARRAY(SELECT DISTINCT unnest(sent_targets || $5::text[])),
  status = CASE WHEN attempts + 1 >= $3 OR $4 THEN 'failed' ELSE 'pending' END,
  next_attempt_at = now() + make_interval(secs => 30 * (attempts + 1) * (attempts + 1))
WHERE id = $1`, id, outcome.Error(), maxAttempts, errors.Is(outcome, ErrPermanent), delivered)
	}
	if err != nil {
		return fmt.Errorf("store: notification %d: %w", id, err)
	}
	return nil
}

// ProcessNotifications claims a batch, calls fn outside any transaction and records each outcome.
// fn returns the targets it delivered (even on partial failure) so retries skip them.
func (s *Store) ProcessNotifications(ctx context.Context, limit, maxAttempts int, fn func(Notification) ([]string, error)) (int, error) {
	items, err := s.ClaimNotifications(ctx, limit, NotificationLease)
	if err != nil {
		return 0, err
	}
	for _, n := range items {
		delivered, ferr := fn(n)
		if err := s.CompleteNotification(ctx, n.ID, delivered, ferr, maxAttempts); err != nil {
			return 0, err
		}
	}
	return len(items), nil
}
