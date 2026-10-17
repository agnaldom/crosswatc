package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TrackedResource mirrors the tracked_resource table (spec §11).
type TrackedResource struct {
	ID                  uuid.UUID
	Name                string
	Kind                string
	Provider            string
	Environment         string
	Owner               string
	RequestedBy         string
	Region              string
	ResourceGroup       string
	ClaimNamespace      string
	ClaimName           string
	ClaimKind           string
	ClaimAPIVersion     string
	Repository          string
	GitPath             string
	ResourceState       string
	CurrentRevision     *int
	CompositionName     string
	CompositionRevision string
	CorrelationID       string
	EverOperational     bool
	Version             int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

const resourceCols = `id, name, kind, provider, environment, owner, coalesce(requested_by,''), coalesce(region,''),
coalesce(resource_group,''), coalesce(claim_namespace,''), coalesce(claim_name,''), coalesce(claim_kind,''),
coalesce(claim_api_version,''), coalesce(repository,''), coalesce(git_path,''), resource_state, current_revision,
coalesce(composition_name,''), coalesce(composition_revision,''), correlation_id, ever_operational, version,
created_at, updated_at`

func scanResource(row pgx.Row) (TrackedResource, error) {
	var r TrackedResource
	err := row.Scan(&r.ID, &r.Name, &r.Kind, &r.Provider, &r.Environment, &r.Owner, &r.RequestedBy, &r.Region,
		&r.ResourceGroup, &r.ClaimNamespace, &r.ClaimName, &r.ClaimKind, &r.ClaimAPIVersion, &r.Repository,
		&r.GitPath, &r.ResourceState, &r.CurrentRevision, &r.CompositionName, &r.CompositionRevision,
		&r.CorrelationID, &r.EverOperational, &r.Version, &r.CreatedAt, &r.UpdatedAt)
	return r, mapErr(err)
}

// UpsertResourceByClaim inserts or refreshes discovery fields of a resource keyed by its Claim.
// ID, correlation_id, created_at, resource_state and ever_operational are preserved on update.
func (s *Store) UpsertResourceByClaim(ctx context.Context, r TrackedResource) (TrackedResource, error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	now := time.Now().UTC()
	if r.ResourceState == "" {
		r.ResourceState = "UNKNOWN"
	}
	row := s.pool.QueryRow(ctx, `
INSERT INTO tracked_resource (id, name, kind, provider, environment, owner, requested_by, region, resource_group,
  claim_namespace, claim_name, claim_kind, claim_api_version, repository, git_path, resource_state, current_revision,
  composition_name, composition_revision, correlation_id, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,nullif($7,''),nullif($8,''),nullif($9,''),$10,$11,$12,$13,nullif($14,''),nullif($15,''),
  $16,$17,nullif($18,''),nullif($19,''),$20,$21,$21)
ON CONFLICT (claim_namespace, claim_name, claim_kind) DO UPDATE SET
  name = excluded.name, kind = excluded.kind, provider = excluded.provider, environment = excluded.environment,
  owner = excluded.owner, requested_by = coalesce(excluded.requested_by, tracked_resource.requested_by),
  region = excluded.region, resource_group = excluded.resource_group, claim_api_version = excluded.claim_api_version,
  repository = excluded.repository, git_path = excluded.git_path,
  current_revision = coalesce(excluded.current_revision, tracked_resource.current_revision),
  composition_name = excluded.composition_name, composition_revision = excluded.composition_revision,
  updated_at = excluded.updated_at, version = tracked_resource.version + 1
RETURNING `+resourceCols,
		r.ID, r.Name, r.Kind, r.Provider, r.Environment, r.Owner, r.RequestedBy, r.Region, r.ResourceGroup,
		r.ClaimNamespace, r.ClaimName, r.ClaimKind, r.ClaimAPIVersion, r.Repository, r.GitPath, r.ResourceState,
		r.CurrentRevision, r.CompositionName, r.CompositionRevision, r.CorrelationID, now)
	return scanResource(row)
}

// UpdateResourceState sets the derived state guarded by optimistic locking.
func (s *Store) UpdateResourceState(ctx context.Context, id uuid.UUID, expectedVersion int64, state string, everOperational bool) (int64, error) {
	var v int64
	err := s.pool.QueryRow(ctx, `
UPDATE tracked_resource SET resource_state = $3, ever_operational = ever_operational OR $4,
  version = version + 1, updated_at = now()
WHERE id = $1 AND version = $2 RETURNING version`, id, expectedVersion, state, everOperational).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrConflict
	}
	return v, mapErr(err)
}

// GetResource returns one resource by ID.
func (s *Store) GetResource(ctx context.Context, id uuid.UUID) (TrackedResource, error) {
	return scanResource(s.pool.QueryRow(ctx, `SELECT `+resourceCols+` FROM tracked_resource WHERE id = $1`, id))
}

// ResourceFilter narrows ListResources (spec §39). Owners == nil means no owner restriction.
type ResourceFilter struct {
	Owners      []string
	Owner       string
	Environment string
	Provider    string
	Kind        string
	State       string
	NamePrefix  string
}

// ListResources returns resources ordered by (name, id) with cursor pagination.
func (s *Store) ListResources(ctx context.Context, f ResourceFilter, p Page) ([]TrackedResource, string, error) {
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
		add("owner = ANY($%d)", f.Owners)
	}
	if f.Owner != "" {
		add("owner = $%d", f.Owner)
	}
	if f.Environment != "" {
		add("environment = $%d", f.Environment)
	}
	if f.Provider != "" {
		add("provider = $%d", f.Provider)
	}
	if f.Kind != "" {
		add("kind = $%d", f.Kind)
	}
	if f.State != "" {
		add("resource_state = $%d", f.State)
	}
	if f.NamePrefix != "" {
		add("name LIKE $%d", escapeLike(f.NamePrefix)+"%")
	}
	if c != nil {
		args = append(args, c.Key, c.ID)
		where = append(where, fmt.Sprintf("(name, id) > ($%d, $%d::uuid)", len(args)-1, len(args)))
	}
	q := `SELECT ` + resourceCols + ` FROM tracked_resource`
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	limit := p.limit()
	args = append(args, limit+1)
	q += fmt.Sprintf(" ORDER BY name, id LIMIT $%d", len(args))

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	var out []TrackedResource
	for rows.Next() {
		r, err := scanResource(rows)
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
		next = encodeCursor(last.Name, last.ID.String())
	}
	return out, next, nil
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
