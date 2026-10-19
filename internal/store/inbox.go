package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrPermanent marks an inbox failure that must not be retried (e.g. malformed payload): wrap it to
// dead-letter the item immediately.
var ErrPermanent = errors.New("store: permanent failure")

// InboxItem mirrors the inbox table (spec §35.1, §37): webhook deliveries queued for the worker.
type InboxItem struct {
	ID         int64
	Source     string
	DeliveryID string
	EventType  string
	Payload    json.RawMessage
	ReceivedAt time.Time
	Attempts   int
}

// InsertInbox queues a delivery. It returns false when (source, delivery_id) was already received,
// so duplicates are acknowledged without reprocessing.
func (s *Store) InsertInbox(ctx context.Context, source, deliveryID, eventType string, payload json.RawMessage) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
INSERT INTO inbox (source, delivery_id, event_type, payload) VALUES ($1, $2, $3, $4)
ON CONFLICT (source, delivery_id) DO NOTHING`, source, deliveryID, eventType, payload)
	if err != nil {
		return false, mapErr(err)
	}
	return tag.RowsAffected() == 1, nil
}

// ProcessInbox claims up to limit pending deliveries with FOR UPDATE SKIP LOCKED, so concurrent workers never
// process the same row, and calls fn for each in arrival order. Successful items are marked processed; failures
// increment attempts and keep last_error. Items reaching maxAttempts are marked processed (dead-lettered) with
// their error preserved, as are items failing with ErrPermanent. It returns the number of items handled.
func (s *Store) ProcessInbox(ctx context.Context, limit, maxAttempts int, fn func(InboxItem) error) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
SELECT id, source, delivery_id, event_type, payload, received_at, attempts FROM inbox
WHERE processed_at IS NULL ORDER BY received_at, id LIMIT $1 FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return 0, err
	}
	var items []InboxItem
	for rows.Next() {
		var it InboxItem
		if err := rows.Scan(&it.ID, &it.Source, &it.DeliveryID, &it.EventType, &it.Payload, &it.ReceivedAt, &it.Attempts); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, it := range items {
		if ferr := fn(it); ferr != nil {
			_, err = tx.Exec(ctx, `
UPDATE inbox SET attempts = attempts + 1, last_error = $2,
  processed_at = CASE WHEN attempts + 1 >= $3 OR $4 THEN now() END
WHERE id = $1`, it.ID, ferr.Error(), maxAttempts, errors.Is(ferr, ErrPermanent))
		} else {
			_, err = tx.Exec(ctx, `UPDATE inbox SET processed_at = now(), attempts = attempts + 1, last_error = NULL WHERE id = $1`, it.ID)
		}
		if err != nil {
			return 0, fmt.Errorf("store: inbox %d: %w", it.ID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(items), nil
}
