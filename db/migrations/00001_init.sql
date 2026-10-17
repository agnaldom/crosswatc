-- +goose Up
CREATE TABLE tracked_resource (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL,
  kind TEXT NOT NULL,
  provider TEXT NOT NULL,
  environment TEXT NOT NULL,
  owner TEXT NOT NULL,
  requested_by TEXT,
  region TEXT,
  resource_group TEXT,
  claim_namespace TEXT,
  claim_name TEXT,
  claim_kind TEXT,
  claim_api_version TEXT,
  repository TEXT,
  git_path TEXT,
  resource_state TEXT NOT NULL DEFAULT 'UNKNOWN',
  current_revision INTEGER,
  composition_name TEXT,
  composition_revision TEXT,
  correlation_id TEXT NOT NULL,
  ever_operational BOOLEAN NOT NULL DEFAULT false,
  version BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (claim_namespace, claim_name, claim_kind),
  UNIQUE (environment, name)
);
CREATE INDEX tracked_resource_owner ON tracked_resource(owner);

CREATE TABLE revision (
  id UUID PRIMARY KEY,
  resource_id UUID NOT NULL REFERENCES tracked_resource(id),
  rev INTEGER NOT NULL,
  kind TEXT NOT NULL,
  state TEXT NOT NULL,
  changes JSONB,
  branch TEXT,
  commit_sha TEXT,
  pr_number INTEGER,
  pr_url TEXT,
  argo_app TEXT,
  failure_reason JSONB,
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  UNIQUE (resource_id, rev)
);
CREATE UNIQUE INDEX revision_one_open_per_resource
ON revision(resource_id)
WHERE state NOT IN ('SUCCEEDED','FAILED','REJECTED','CANCELLED','SUPERSEDED');

CREATE TABLE operation (
  id UUID PRIMARY KEY,
  resource_id UUID NOT NULL REFERENCES tracked_resource(id),
  revision_id UUID REFERENCES revision(id),
  correlation_id TEXT UNIQUE NOT NULL,
  type TEXT NOT NULL,
  status TEXT NOT NULL,
  requested_by TEXT NOT NULL,
  requested_at TIMESTAMPTZ NOT NULL,
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ
);
CREATE INDEX operation_resource ON operation(resource_id, requested_at DESC);

CREATE TABLE tracker_event (
  id UUID PRIMARY KEY,
  dedupe_key TEXT UNIQUE NOT NULL,
  resource_id UUID,
  revision_id UUID,
  operation_id UUID,
  correlation_id TEXT,
  source TEXT NOT NULL,
  type TEXT NOT NULL,
  level TEXT NOT NULL,
  message TEXT,
  raw_payload JSONB,
  created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX tracker_event_resource_ts ON tracker_event(resource_id, created_at DESC);

CREATE TABLE resource_projection (
  resource_id UUID PRIMARY KEY REFERENCES tracked_resource(id),
  desired_state JSONB,
  applied_state JSONB,
  observed_state JSONB,
  drift_state JSONB,
  stages JSONB,
  health JSONB,
  last_reconcile_at TIMESTAMPTZ,
  version BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE resource_topology (
  resource_id UUID PRIMARY KEY REFERENCES tracked_resource(id),
  nodes JSONB NOT NULL,
  edges JSONB NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE ingest_checkpoint (
  source TEXT PRIMARY KEY,
  cursor TEXT,
  last_success_at TIMESTAMPTZ,
  last_error TEXT,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE inbox (
  id BIGSERIAL PRIMARY KEY,
  source TEXT NOT NULL,
  delivery_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  payload JSONB NOT NULL,
  received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  processed_at TIMESTAMPTZ,
  attempts INTEGER NOT NULL DEFAULT 0,
  last_error TEXT,
  UNIQUE (source, delivery_id)
);
CREATE INDEX inbox_pending ON inbox(received_at) WHERE processed_at IS NULL;

CREATE TABLE audit_log (
  id BIGSERIAL PRIMARY KEY,
  actor TEXT NOT NULL,
  action TEXT NOT NULL,
  resource_id UUID,
  revision_id UUID,
  operation_id UUID,
  correlation_id TEXT,
  before JSONB,
  after JSONB,
  pr_url TEXT,
  commit_sha TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Append-only roles (§37). NOLOGIN groups; deployment grants membership to login users.
-- +goose StatementBegin
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'crosswatc_app') THEN
    CREATE ROLE crosswatc_app NOLOGIN;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'crosswatc_maintenance') THEN
    CREATE ROLE crosswatc_maintenance NOLOGIN;
  END IF;
END
$$;
-- +goose StatementEnd

GRANT SELECT, INSERT, UPDATE, DELETE ON tracked_resource, revision, operation,
  resource_projection, resource_topology, ingest_checkpoint, inbox TO crosswatc_app;
GRANT SELECT, INSERT ON tracker_event, audit_log TO crosswatc_app;
GRANT USAGE ON SEQUENCE inbox_id_seq, audit_log_id_seq TO crosswatc_app;
GRANT SELECT, UPDATE (raw_payload) ON tracker_event TO crosswatc_maintenance;

-- +goose Down
DROP TABLE audit_log, inbox, ingest_checkpoint, resource_topology, resource_projection,
  tracker_event, operation, revision, tracked_resource;
