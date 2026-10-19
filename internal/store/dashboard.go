package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Dashboard aggregates the home page (totals, attention, open revisions, recent events).
type Dashboard struct {
	ByState       map[string]int
	Attention     []AttentionItem
	OpenRevisions []OpenRevision
	RecentEvents  []DashboardEvent
}

// AttentionItem is a resource that is degraded, failed or drifted.
type AttentionItem struct {
	ID     uuid.UUID
	Name   string
	Kind   string
	State  string
	Drift  bool
	Synced *bool
	Ready  *bool
}

// OpenRevision is a non-terminal revision (change in progress).
type OpenRevision struct {
	ResourceID   uuid.UUID
	ResourceName string
	Rev          int
	Kind         string
	State        string
	PRURL        string
	StartedAt    *time.Time
}

// DashboardEvent is a tracker event with its resource name.
type DashboardEvent struct {
	Event
	ResourceName string
}

const dashboardLimit = 20

// GetDashboard builds the dashboard; owners == nil means no owner restriction (SRE).
func (s *Store) GetDashboard(ctx context.Context, owners []string) (Dashboard, error) {
	// $1 IS NULL disables the owner filter (SRE); otherwise owner = ANY($1).
	var scope any
	if owners != nil {
		scope = owners
	}
	d := Dashboard{ByState: map[string]int{}}

	rows, err := s.pool.Query(ctx, `SELECT resource_state, count(*) FROM tracked_resource
WHERE $1::text[] IS NULL OR owner = ANY($1) GROUP BY 1`, scope)
	if err != nil {
		return d, err
	}
	for rows.Next() {
		var st string
		var n int
		if err := rows.Scan(&st, &n); err != nil {
			rows.Close()
			return d, err
		}
		d.ByState[st] = n
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return d, err
	}

	rows, err = s.pool.Query(ctx, `
SELECT t.id, t.name, t.kind, t.resource_state, coalesce((p.drift_state->>'detected')::boolean, false),
  (p.observed_state->>'synced')::boolean, (p.observed_state->>'ready')::boolean
FROM tracked_resource t LEFT JOIN resource_projection p ON p.resource_id = t.id
WHERE ($1::text[] IS NULL OR t.owner = ANY($1))
  AND (t.resource_state IN ('DEGRADED','FAILED') OR coalesce((p.drift_state->>'detected')::boolean, false)
       OR (t.resource_state = 'OPERATIONAL' AND (p.observed_state->>'ready')::boolean = false))
ORDER BY t.updated_at DESC LIMIT $2`, scope, dashboardLimit)
	if err != nil {
		return d, err
	}
	for rows.Next() {
		var a AttentionItem
		if err := rows.Scan(&a.ID, &a.Name, &a.Kind, &a.State, &a.Drift, &a.Synced, &a.Ready); err != nil {
			rows.Close()
			return d, err
		}
		d.Attention = append(d.Attention, a)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return d, err
	}

	rows, err = s.pool.Query(ctx, `
SELECT r.resource_id, t.name, r.rev, r.kind, r.state, coalesce(r.pr_url,''), r.started_at
FROM revision r JOIN tracked_resource t ON t.id = r.resource_id
WHERE ($1::text[] IS NULL OR t.owner = ANY($1))
  AND r.state NOT IN ('SUCCEEDED','FAILED','REJECTED','CANCELLED','SUPERSEDED')
ORDER BY r.started_at DESC NULLS LAST LIMIT $2`, scope, dashboardLimit)
	if err != nil {
		return d, err
	}
	for rows.Next() {
		var o OpenRevision
		if err := rows.Scan(&o.ResourceID, &o.ResourceName, &o.Rev, &o.Kind, &o.State, &o.PRURL, &o.StartedAt); err != nil {
			rows.Close()
			return d, err
		}
		d.OpenRevisions = append(d.OpenRevisions, o)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return d, err
	}

	rows, err = s.pool.Query(ctx, `
SELECT e.id, e.resource_id, e.revision_id, coalesce(e.correlation_id,''), e.source, e.type, e.level, coalesce(e.message,''),
  e.created_at, t.name
FROM tracker_event e JOIN tracked_resource t ON t.id = e.resource_id
WHERE ($1::text[] IS NULL OR t.owner = ANY($1))
ORDER BY e.created_at DESC LIMIT $2`, scope, 15)
	if err != nil {
		return d, err
	}
	defer rows.Close()
	for rows.Next() {
		var e DashboardEvent
		if err := rows.Scan(&e.ID, &e.ResourceID, &e.RevisionID, &e.CorrelationID, &e.Source, &e.Type, &e.Level,
			&e.Message, &e.CreatedAt, &e.ResourceName); err != nil {
			return d, err
		}
		d.RecentEvents = append(d.RecentEvents, e)
	}
	return d, rows.Err()
}
