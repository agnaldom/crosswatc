package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// StreamChannel is the LISTEN/NOTIFY channel of realtime events (spec §45).
const StreamChannel = "crosswatc_stream"

// StreamEvent mirrors stream_event: one SSE message, written by database triggers.
type StreamEvent struct {
	ID         int64
	Type       string
	ResourceID *uuid.UUID
	Owner      string
	Data       json.RawMessage
	CreatedAt  time.Time
}

// StreamEventsAfter returns events with id > afterID created after since, oldest first.
func (s *Store) StreamEventsAfter(ctx context.Context, afterID int64, since time.Time, limit int) ([]StreamEvent, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, type, resource_id, coalesce(owner,''), data, created_at FROM stream_event
WHERE id > $1 AND created_at >= $2 ORDER BY id LIMIT $3`, afterID, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StreamEvent
	for rows.Next() {
		var e StreamEvent
		if err := rows.Scan(&e.ID, &e.Type, &e.ResourceID, &e.Owner, &e.Data, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// LatestStreamID returns the highest stream event id (0 when empty).
func (s *Store) LatestStreamID(ctx context.Context) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `SELECT coalesce(max(id), 0) FROM stream_event`).Scan(&id)
	return id, err
}

// PruneStream deletes stream events older than olderThan (the replay buffer only needs minutes).
func (s *Store) PruneStream(ctx context.Context, olderThan time.Duration) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM stream_event WHERE created_at < now() - make_interval(secs => $1)`, olderThan.Seconds())
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// Listen holds a dedicated connection LISTENing on channel and calls onNotify for each notification until ctx
// ends or the connection fails. onConnect runs once the LISTEN is active (callers catch up there).
func (s *Store) Listen(ctx context.Context, channel string, onConnect func(), onNotify func(payload string)) error {
	pc, err := s.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	conn := pc.Hijack() // never return a LISTENing connection to the pool
	defer func() { _ = conn.Close(context.Background()) }()
	if _, err := conn.Exec(ctx, "LISTEN "+pgx.Identifier{channel}.Sanitize()); err != nil {
		return err
	}
	if onConnect != nil {
		onConnect()
	}
	for {
		n, err := conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		onNotify(n.Payload)
	}
}
