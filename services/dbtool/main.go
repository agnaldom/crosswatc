// Command crosswatc-dbtool runs migrations and generates synthetic data for load tests.
//
//	crosswatc-dbtool migrate
//	crosswatc-dbtool seed-load -resources 2000 -squads 20 -events 20
//
// Reads CROSSWATC_DATABASE_URL. seed-load refuses to run when CROSSWATC_ENV=production.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"

	"github.com/agnaldom/crosswatc/internal/store"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	url := os.Getenv("CROSSWATC_DATABASE_URL")
	if url == "" || len(args) == 0 {
		return errors.New("usage: CROSSWATC_DATABASE_URL=... crosswatc-dbtool migrate|seed-load [flags]")
	}
	switch args[0] {
	case "migrate":
		return store.Migrate(ctx, url)
	case "seed-load":
		if os.Getenv("CROSSWATC_ENV") == "production" {
			return errors.New("seed-load is disabled in production")
		}
		fs := flag.NewFlagSet("seed-load", flag.ContinueOnError)
		n := fs.Int("resources", 2000, "resources to create")
		squads := fs.Int("squads", 20, "distinct owners (load0..loadN-1)")
		events := fs.Int("events", 20, "events per resource")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		return seed(ctx, url, *n, *squads, *events)
	}
	return fmt.Errorf("unknown command %q", args[0])
}

// seed inserts synthetic resources with projection, topology, revisions and events using set-based SQL.
func seed(ctx context.Context, url string, n, squads, events int) error {
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close(ctx) }()
	_, err = conn.Exec(ctx, seedSQL, n, squads, events)
	if err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	var count int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM tracked_resource WHERE name LIKE 'load-%'`).Scan(&count); err != nil {
		return err
	}
	fmt.Printf("seeded: %d load resources\n", count)
	return nil
}

const seedSQL = `
WITH params AS (SELECT $1::int AS n, $2::int AS squads, $3::int AS events),
res AS (
  INSERT INTO tracked_resource (id, name, kind, provider, environment, owner, claim_namespace, claim_name, claim_kind,
    claim_api_version, repository, git_path, resource_state, current_revision, composition_name, correlation_id,
    ever_operational, created_at, updated_at)
  SELECT gen_random_uuid(), 'load-' || i, (ARRAY['PostgreSQLInstance','VirtualMachine','StorageAccount'])[1 + i % 3], 'azure',
    (ARRAY['dev','hlg','prd'])[1 + i % 3], 'load' || (i % p.squads), 'team-load' || (i % p.squads), 'load-' || i,
    'LoadClaim', 'load.crosswatc.io/v1', 'org/claims', 'claims/load-' || i || '.yaml',
    (ARRAY['OPERATIONAL','OPERATIONAL','OPERATIONAL','PROVISIONING','DEGRADED'])[1 + i % 5], 1, 'load-v1',
    'crosswatc-req-' || upper(substr(md5(i::text), 1, 26)), true, now() - (i || ' minutes')::interval, now()
  FROM params p, generate_series(1, p.n) AS i
  ON CONFLICT DO NOTHING
  RETURNING id, correlation_id
),
proj AS (
  INSERT INTO resource_projection (resource_id, applied_state, drift_state, stages, health, last_reconcile_at)
  SELECT id, '{"syncedWithGit":true,"spec":{"parameters":{"vmSize":"Standard_D2as_v5"}}}', '{"detected":false}',
    '[{"stage":"REQUEST","status":"SKIPPED"},{"stage":"PULL_REQUEST","status":"SUCCEEDED"},{"stage":"CI_POLICIES","status":"SUCCEEDED"},{"stage":"ARGO_CD","status":"SUCCEEDED"},{"stage":"CROSSPLANE","status":"SUCCEEDED"},{"stage":"CLOUD","status":"SUCCEEDED"},{"stage":"OPERATIONAL","status":"SUCCEEDED"}]',
    '{"crossplane":"ok","argocd":"ok","cloud":"ok","configuration":"ok","secrets":"ok"}', now()
  FROM res RETURNING resource_id
),
topo AS (
  INSERT INTO resource_topology (resource_id, nodes, edges, updated_at)
  SELECT id, '[{"id":"claim","type":"claim","kind":"LoadClaim","name":"c","status":"ready"},{"id":"xr","type":"composite","kind":"XLoad","name":"x","status":"ready"},{"id":"mr","type":"managed-resource","kind":"Server","name":"m","status":"ready","provider":"azure"}]',
    '[{"source":"claim","target":"xr"},{"source":"xr","target":"mr"}]', now()
  FROM res RETURNING resource_id
),
rev AS (
  INSERT INTO revision (id, resource_id, rev, kind, state, started_at, completed_at, state_changed_at)
  SELECT gen_random_uuid(), id, 1, 'CREATE', 'SUCCEEDED', now() - interval '1 day', now() - interval '23 hours', now()
  FROM res RETURNING resource_id
)
INSERT INTO tracker_event (id, dedupe_key, resource_id, correlation_id, source, type, level, message, created_at)
SELECT gen_random_uuid(), 'load:' || r.id || ':' || e, r.id, r.correlation_id, 'crosswatc', 'LIFECYCLE.OPERATIONAL', 'info',
  'evento sintético ' || e, now() - (e || ' minutes')::interval
FROM res r, params p, generate_series(1, p.events) AS e
`
