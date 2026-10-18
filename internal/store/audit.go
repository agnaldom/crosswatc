package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AuditEntry mirrors audit_log (spec §50). Append-only.
type AuditEntry struct {
	ID            int64
	Actor         string
	Action        string
	ResourceID    *uuid.UUID
	RevisionID    *uuid.UUID
	OperationID   *uuid.UUID
	CorrelationID string
	Before        json.RawMessage
	After         json.RawMessage
	PRURL         string
	CommitSHA     string
	CreatedAt     time.Time
}

// InsertAudit appends an audit record.
func (s *Store) InsertAudit(ctx context.Context, a AuditEntry) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO audit_log (actor, action, resource_id, revision_id, operation_id, correlation_id, before, after, pr_url, commit_sha)
VALUES ($1,$2,$3,$4,$5,nullif($6,''),$7,$8,nullif($9,''),nullif($10,''))`,
		a.Actor, a.Action, a.ResourceID, a.RevisionID, a.OperationID, a.CorrelationID, nullJSON(a.Before), nullJSON(a.After), a.PRURL, a.CommitSHA)
	return mapErr(err)
}

// AuditFilter narrows ListAudit. Owners == nil means unrestricted.
type AuditFilter struct {
	Owners     []string
	ResourceID *uuid.UUID
	Actor      string
	Action     string
}

// ListAudit returns audit entries newest first with cursor pagination on id.
func (s *Store) ListAudit(ctx context.Context, f AuditFilter, p Page) ([]AuditEntry, string, error) {
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
		add("tr.owner = ANY($%d)", f.Owners)
	}
	if f.ResourceID != nil {
		add("a.resource_id = $%d", *f.ResourceID)
	}
	if f.Actor != "" {
		add("a.actor = $%d", f.Actor)
	}
	if f.Action != "" {
		add("a.action = $%d", f.Action)
	}
	if c != nil {
		id, err := strconv.ParseInt(c.ID, 10, 64)
		if err != nil {
			return nil, "", ErrBadCursor
		}
		add("a.id < $%d", id)
	}
	q := `SELECT a.id, a.actor, a.action, a.resource_id, a.revision_id, a.operation_id, coalesce(a.correlation_id,''),
a.before, a.after, coalesce(a.pr_url,''), coalesce(a.commit_sha,''), a.created_at
FROM audit_log a LEFT JOIN tracked_resource tr ON tr.id = a.resource_id`
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	limit := p.limit()
	args = append(args, limit+1)
	q += fmt.Sprintf(" ORDER BY a.id DESC LIMIT $%d", len(args))
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	var out []AuditEntry
	for rows.Next() {
		var a AuditEntry
		if err := rows.Scan(&a.ID, &a.Actor, &a.Action, &a.ResourceID, &a.RevisionID, &a.OperationID, &a.CorrelationID,
			&a.Before, &a.After, &a.PRURL, &a.CommitSHA, &a.CreatedAt); err != nil {
			return nil, "", err
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(out) > limit {
		out = out[:limit]
		next = encodeCursor("", strconv.FormatInt(out[limit-1].ID, 10))
	}
	return out, next, nil
}
