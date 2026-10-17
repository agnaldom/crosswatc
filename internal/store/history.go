package store

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ListRevisionsPage returns revisions of a resource, newest first, with cursor pagination (spec §39).
func (s *Store) ListRevisionsPage(ctx context.Context, resourceID uuid.UUID, p Page) ([]Revision, string, error) {
	c, err := decodeCursor(p.Cursor)
	if err != nil {
		return nil, "", err
	}
	limit := p.limit()
	args := []any{resourceID, limit + 1}
	q := `SELECT ` + revisionCols + ` FROM revision WHERE resource_id = $1`
	if c != nil {
		rev, err := strconv.Atoi(c.Key)
		if err != nil {
			return nil, "", ErrBadCursor
		}
		args = append(args, rev)
		q += ` AND rev < $3`
	}
	q += ` ORDER BY rev DESC LIMIT $2`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	var out []Revision
	for rows.Next() {
		r, err := scanRevision(rows)
		if err != nil {
			return nil, "", err
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(out) > limit {
		out = out[:limit]
		last := out[limit-1]
		next = encodeCursor(strconv.Itoa(last.Rev), last.ID.String())
	}
	return out, next, nil
}

// GetRevision returns revision number rev of a resource.
func (s *Store) GetRevision(ctx context.Context, resourceID uuid.UUID, rev int) (Revision, error) {
	return scanRevision(s.pool.QueryRow(ctx, `SELECT `+revisionCols+` FROM revision WHERE resource_id = $1 AND rev = $2`, resourceID, rev))
}

// OperationFilter narrows ListOperationsPage. Owners == nil means no owner restriction (SRE).
type OperationFilter struct {
	Owners     []string
	ResourceID *uuid.UUID
	RevisionID *uuid.UUID
	Status     string
	Type       string
}

const operationCols = `o.id, o.resource_id, o.revision_id, o.correlation_id, o.type, o.status, o.requested_by, o.requested_at,
o.started_at, o.finished_at, r.name, r.owner`

func scanOperation(row interface{ Scan(...any) error }) (Operation, error) {
	var o Operation
	err := row.Scan(&o.ID, &o.ResourceID, &o.RevisionID, &o.CorrelationID, &o.Type, &o.Status, &o.RequestedBy, &o.RequestedAt,
		&o.StartedAt, &o.FinishedAt, &o.ResourceName, &o.Owner)
	return o, mapErr(err)
}

// ListOperationsPage returns operations newest first (requested_at, id), with the owning resource's name and owner.
func (s *Store) ListOperationsPage(ctx context.Context, f OperationFilter, p Page) ([]Operation, string, error) {
	c, err := decodeCursor(p.Cursor)
	if err != nil {
		return nil, "", err
	}
	var (
		where []string
		args  []any
	)
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}
	if f.Owners != nil {
		add("r.owner = ANY($%d)", f.Owners)
	}
	if f.ResourceID != nil {
		add("o.resource_id = $%d", *f.ResourceID)
	}
	if f.RevisionID != nil {
		add("o.revision_id = $%d", *f.RevisionID)
	}
	if f.Status != "" {
		add("o.status = $%d", f.Status)
	}
	if f.Type != "" {
		add("o.type = $%d", f.Type)
	}
	if c != nil {
		ts, err := time.Parse(time.RFC3339Nano, c.Key)
		if err != nil {
			return nil, "", ErrBadCursor
		}
		args = append(args, ts, c.ID)
		where = append(where, fmt.Sprintf("(o.requested_at, o.id) < ($%d, $%d::uuid)", len(args)-1, len(args)))
	}
	q := `SELECT ` + operationCols + ` FROM operation o JOIN tracked_resource r ON r.id = o.resource_id`
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	limit := p.limit()
	args = append(args, limit+1)
	q += fmt.Sprintf(" ORDER BY o.requested_at DESC, o.id DESC LIMIT $%d", len(args))
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	var out []Operation
	for rows.Next() {
		o, err := scanOperation(rows)
		if err != nil {
			return nil, "", err
		}
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(out) > limit {
		out = out[:limit]
		last := out[limit-1]
		next = encodeCursor(last.RequestedAt.Format(time.RFC3339Nano), last.ID.String())
	}
	return out, next, nil
}

// GetOperation returns one operation with its resource's name and owner.
func (s *Store) GetOperation(ctx context.Context, id uuid.UUID) (Operation, error) {
	return scanOperation(s.pool.QueryRow(ctx, `SELECT `+operationCols+`
FROM operation o JOIN tracked_resource r ON r.id = o.resource_id WHERE o.id = $1`, id))
}

// ListEventsByRevision returns the events of a revision in chronological order (spec §46 Events).
func (s *Store) ListEventsByRevision(ctx context.Context, revisionID uuid.UUID, limit int) ([]Event, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, dedupe_key, resource_id, revision_id, operation_id, coalesce(correlation_id,''),
source, type, level, coalesce(message,''), created_at FROM tracker_event WHERE revision_id = $1
ORDER BY created_at, id LIMIT $2`, revisionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.DedupeKey, &e.ResourceID, &e.RevisionID, &e.OperationID, &e.CorrelationID,
			&e.Source, &e.Type, &e.Level, &e.Message, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
