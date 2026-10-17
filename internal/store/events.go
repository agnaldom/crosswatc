package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Event mirrors tracker_event (spec §34). Append-only.
type Event struct {
	ID            uuid.UUID
	DedupeKey     string
	ResourceID    *uuid.UUID
	RevisionID    *uuid.UUID
	OperationID   *uuid.UUID
	CorrelationID string
	Source        string
	Type          string
	Level         string
	Message       string
	RawPayload    json.RawMessage
	CreatedAt     time.Time
}

// InsertEvent appends an event. Returns false when dedupe_key already exists.
func (s *Store) InsertEvent(ctx context.Context, e Event) (bool, error) {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	tag, err := s.pool.Exec(ctx, `
INSERT INTO tracker_event (id, dedupe_key, resource_id, revision_id, operation_id, correlation_id, source, type, level, message, raw_payload, created_at)
VALUES ($1,$2,$3,$4,$5,nullif($6,''),$7,$8,$9,nullif($10,''),$11,$12)
ON CONFLICT (dedupe_key) DO NOTHING`,
		e.ID, e.DedupeKey, e.ResourceID, e.RevisionID, e.OperationID, e.CorrelationID, e.Source, e.Type, e.Level,
		e.Message, nullJSON(e.RawPayload), e.CreatedAt)
	if err != nil {
		return false, mapErr(err)
	}
	return tag.RowsAffected() == 1, nil
}

// ListEvents returns events of a resource, newest first.
func (s *Store) ListEvents(ctx context.Context, resourceID uuid.UUID, p Page) ([]Event, string, error) {
	c, err := decodeCursor(p.Cursor)
	if err != nil {
		return nil, "", err
	}
	limit := p.limit()
	args := []any{resourceID, limit + 1}
	q := `SELECT id, dedupe_key, resource_id, revision_id, operation_id, coalesce(correlation_id,''), source, type,
level, coalesce(message,''), created_at FROM tracker_event WHERE resource_id = $1`
	if c != nil {
		ts, err := time.Parse(time.RFC3339Nano, c.Key)
		if err != nil {
			return nil, "", ErrBadCursor
		}
		args = append(args, ts, c.ID)
		q += ` AND (created_at, id) < ($3, $4::uuid)`
	}
	q += ` ORDER BY created_at DESC, id DESC LIMIT $2`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.DedupeKey, &e.ResourceID, &e.RevisionID, &e.OperationID, &e.CorrelationID,
			&e.Source, &e.Type, &e.Level, &e.Message, &e.CreatedAt); err != nil {
			return nil, "", err
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(out) > limit {
		out = out[:limit]
		last := out[limit-1]
		next = encodeCursor(last.CreatedAt.Format(time.RFC3339Nano), last.ID.String())
	}
	return out, next, nil
}
