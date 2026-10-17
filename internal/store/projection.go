package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Projection mirrors resource_projection (spec §18, §19, §31).
type Projection struct {
	ResourceID      uuid.UUID
	DesiredState    json.RawMessage
	AppliedState    json.RawMessage
	ObservedState   json.RawMessage
	DriftState      json.RawMessage
	Stages          json.RawMessage
	Health          json.RawMessage
	LastReconcileAt *time.Time
	Version         int64
}

// UpsertProjection replaces the projection of a resource.
func (s *Store) UpsertProjection(ctx context.Context, p Projection) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO resource_projection (resource_id, desired_state, applied_state, observed_state, drift_state, stages, health, last_reconcile_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (resource_id) DO UPDATE SET desired_state = excluded.desired_state, applied_state = excluded.applied_state,
  observed_state = excluded.observed_state, drift_state = excluded.drift_state, stages = excluded.stages,
  health = excluded.health, last_reconcile_at = excluded.last_reconcile_at, version = resource_projection.version + 1`,
		p.ResourceID, nullJSON(p.DesiredState), nullJSON(p.AppliedState), nullJSON(p.ObservedState),
		nullJSON(p.DriftState), nullJSON(p.Stages), nullJSON(p.Health), p.LastReconcileAt)
	return mapErr(err)
}

// GetProjection returns the projection of a resource.
func (s *Store) GetProjection(ctx context.Context, id uuid.UUID) (Projection, error) {
	p := Projection{ResourceID: id}
	err := s.pool.QueryRow(ctx, `
SELECT desired_state, applied_state, observed_state, drift_state, stages, health, last_reconcile_at, version
FROM resource_projection WHERE resource_id = $1`, id).Scan(&p.DesiredState, &p.AppliedState, &p.ObservedState,
		&p.DriftState, &p.Stages, &p.Health, &p.LastReconcileAt, &p.Version)
	return p, mapErr(err)
}

// Topology mirrors resource_topology (spec §29).
type Topology struct {
	ResourceID uuid.UUID
	Nodes      json.RawMessage
	Edges      json.RawMessage
	UpdatedAt  time.Time
}

// UpsertTopology replaces the topology graph of a resource.
func (s *Store) UpsertTopology(ctx context.Context, t Topology) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO resource_topology (resource_id, nodes, edges, updated_at) VALUES ($1,$2,$3,now())
ON CONFLICT (resource_id) DO UPDATE SET nodes = excluded.nodes, edges = excluded.edges, updated_at = now()`,
		t.ResourceID, t.Nodes, t.Edges)
	return mapErr(err)
}

// GetTopology returns the topology graph of a resource.
func (s *Store) GetTopology(ctx context.Context, id uuid.UUID) (Topology, error) {
	t := Topology{ResourceID: id}
	err := s.pool.QueryRow(ctx, `SELECT nodes, edges, updated_at FROM resource_topology WHERE resource_id = $1`, id).
		Scan(&t.Nodes, &t.Edges, &t.UpdatedAt)
	return t, mapErr(err)
}

func nullJSON(b json.RawMessage) any {
	if len(b) == 0 {
		return nil
	}
	return b
}
