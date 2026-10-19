package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Revision mirrors the revision table (spec §13).
type Revision struct {
	ID             uuid.UUID
	ResourceID     uuid.UUID
	Rev            int
	Kind           string
	State          string
	Changes        json.RawMessage
	Branch         string
	CommitSHA      string
	PRNumber       *int
	PRURL          string
	ArgoApp        string
	FailureReason  json.RawMessage
	StartedAt      *time.Time
	CompletedAt    *time.Time
	StateChangedAt time.Time
}

const revisionCols = `id, resource_id, rev, kind, state, changes, coalesce(branch,''), coalesce(commit_sha,''),
pr_number, coalesce(pr_url,''), coalesce(argo_app,''), failure_reason, started_at, completed_at, state_changed_at`

func scanRevision(row pgx.Row) (Revision, error) {
	var r Revision
	err := row.Scan(&r.ID, &r.ResourceID, &r.Rev, &r.Kind, &r.State, &r.Changes, &r.Branch, &r.CommitSHA,
		&r.PRNumber, &r.PRURL, &r.ArgoApp, &r.FailureReason, &r.StartedAt, &r.CompletedAt, &r.StateChangedAt)
	return r, mapErr(err)
}

// InsertRevision creates a revision. Returns ErrConflict if another open revision exists (§14.1).
func (s *Store) InsertRevision(ctx context.Context, r Revision) (Revision, error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return scanRevision(s.pool.QueryRow(ctx, `
INSERT INTO revision (id, resource_id, rev, kind, state, changes, branch, commit_sha, pr_number, pr_url, argo_app, failure_reason, started_at, completed_at)
VALUES ($1,$2,$3,$4,$5,$6,nullif($7,''),nullif($8,''),$9,nullif($10,''),nullif($11,''),$12,$13,$14)
RETURNING `+revisionCols,
		r.ID, r.ResourceID, r.Rev, r.Kind, r.State, nullJSON(r.Changes), r.Branch, r.CommitSHA, r.PRNumber, r.PRURL,
		r.ArgoApp, nullJSON(r.FailureReason), r.StartedAt, r.CompletedAt))
}

// UpdateRevision persists mutable fields (state, git/argo refs, failure, completion).
// state_changed_at is bumped only when the state actually changes.
func (s *Store) UpdateRevision(ctx context.Context, r Revision) error {
	tag, err := s.pool.Exec(ctx, `
UPDATE revision SET state = $2, branch = nullif($3,''), commit_sha = nullif($4,''), pr_number = $5,
  pr_url = nullif($6,''), argo_app = nullif($7,''), failure_reason = $8, completed_at = $9,
  state_changed_at = CASE WHEN state <> $2 THEN now() ELSE state_changed_at END
WHERE id = $1`, r.ID, r.State, r.Branch, r.CommitSHA, r.PRNumber, r.PRURL, r.ArgoApp, nullJSON(r.FailureReason), r.CompletedAt)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListRevisions returns revisions of a resource, newest first.
func (s *Store) ListRevisions(ctx context.Context, resourceID uuid.UUID) ([]Revision, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+revisionCols+` FROM revision WHERE resource_id = $1 ORDER BY rev DESC`, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Revision
	for rows.Next() {
		r, err := scanRevision(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Operation mirrors the operation table (spec §15).
type Operation struct {
	ID            uuid.UUID
	ResourceID    uuid.UUID
	RevisionID    *uuid.UUID
	CorrelationID string
	Type          string
	Status        string
	RequestedBy   string
	RequestedAt   time.Time
	StartedAt     *time.Time
	FinishedAt    *time.Time

	// Read-side only (ListOperationsPage/GetOperation): the owning resource.
	ResourceName string
	Owner        string
}

// InsertOperation creates an operation.
func (s *Store) InsertOperation(ctx context.Context, o Operation) (Operation, error) {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	if o.RequestedAt.IsZero() {
		o.RequestedAt = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx, `
INSERT INTO operation (id, resource_id, revision_id, correlation_id, type, status, requested_by, requested_at, started_at, finished_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, o.ID, o.ResourceID, o.RevisionID, o.CorrelationID, o.Type, o.Status,
		o.RequestedBy, o.RequestedAt, o.StartedAt, o.FinishedAt)
	return o, mapErr(err)
}

// ListOperations returns operations of a resource, newest first.
func (s *Store) ListOperations(ctx context.Context, resourceID uuid.UUID) ([]Operation, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, resource_id, revision_id, correlation_id, type, status, requested_by, requested_at, started_at, finished_at
FROM operation WHERE resource_id = $1 ORDER BY requested_at DESC`, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Operation
	for rows.Next() {
		var o Operation
		if err := rows.Scan(&o.ID, &o.ResourceID, &o.RevisionID, &o.CorrelationID, &o.Type, &o.Status,
			&o.RequestedBy, &o.RequestedAt, &o.StartedAt, &o.FinishedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// UpdateOperationsForRevision sets the status of operations tied to a revision.
func (s *Store) UpdateOperationsForRevision(ctx context.Context, revisionID uuid.UUID, status string, finishedAt *time.Time) error {
	_, err := s.pool.Exec(ctx, `
UPDATE operation SET status = $2, started_at = coalesce(started_at, now()), finished_at = coalesce($3, finished_at)
WHERE revision_id = $1`, revisionID, status, finishedAt)
	return mapErr(err)
}

// SaveCheckpoint records ingestion progress for a source.
func (s *Store) SaveCheckpoint(ctx context.Context, source, cur string, ingestErr error) error {
	var msg *string
	if ingestErr != nil {
		m := ingestErr.Error()
		msg = &m
	}
	_, err := s.pool.Exec(ctx, `
INSERT INTO ingest_checkpoint (source, cursor, last_success_at, last_error, updated_at)
VALUES ($1, nullif($2,''), CASE WHEN $3::text IS NULL THEN now() END, $3, now())
ON CONFLICT (source) DO UPDATE SET cursor = coalesce(excluded.cursor, ingest_checkpoint.cursor),
  last_success_at = coalesce(excluded.last_success_at, ingest_checkpoint.last_success_at),
  last_error = excluded.last_error, updated_at = now()`, source, cur, msg)
	return mapErr(err)
}
