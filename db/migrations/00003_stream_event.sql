-- +goose Up
-- Realtime fan-out (spec §45): every relevant change of the read model appends a row here and
-- NOTIFYs crosswatc_stream. API replicas LISTEN, read rows by id and push them over SSE; the table is
-- also the Last-Event-ID replay buffer (ids are monotonic; rows are pruned after a few minutes).
CREATE TABLE stream_event (
  id BIGSERIAL PRIMARY KEY,
  type TEXT NOT NULL,
  resource_id UUID,
  owner TEXT,
  data JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX stream_event_created ON stream_event(created_at);

-- +goose StatementBegin
CREATE FUNCTION crosswatc_stream_publish(p_type TEXT, p_resource UUID, p_data JSONB) RETURNS void AS $$
DECLARE
  v_id BIGINT;
  v_owner TEXT;
BEGIN
  SELECT owner INTO v_owner FROM tracked_resource WHERE id = p_resource;
  INSERT INTO stream_event (type, resource_id, owner, data) VALUES (p_type, p_resource, v_owner, p_data)
  RETURNING id INTO v_id;
  PERFORM pg_notify('crosswatc_stream', v_id::text);
END
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- resource.updated: state/identity changes (the resync bumps version/updated_at only: ignored).
-- +goose StatementBegin
CREATE FUNCTION crosswatc_stream_resource() RETURNS trigger AS $$
BEGIN
  IF TG_OP = 'UPDATE' AND (to_jsonb(OLD) - 'updated_at' - 'version') = (to_jsonb(NEW) - 'updated_at' - 'version') THEN
    RETURN NULL;
  END IF;
  PERFORM crosswatc_stream_publish('resource.updated', NEW.id, jsonb_build_object(
    'resourceId', NEW.id, 'name', NEW.name, 'state', NEW.resource_state, 'currentRevision', NEW.current_revision));
  RETURN NULL;
END
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
CREATE TRIGGER stream_resource AFTER INSERT OR UPDATE ON tracked_resource
FOR EACH ROW EXECUTE FUNCTION crosswatc_stream_resource();

-- +goose StatementBegin
CREATE FUNCTION crosswatc_stream_revision() RETURNS trigger AS $$
BEGIN
  IF TG_OP = 'UPDATE' AND (to_jsonb(OLD) - 'state_changed_at') = (to_jsonb(NEW) - 'state_changed_at') THEN
    RETURN NULL;
  END IF;
  PERFORM crosswatc_stream_publish('revision.updated', NEW.resource_id, jsonb_build_object(
    'resourceId', NEW.resource_id, 'revisionId', NEW.id, 'revision', NEW.rev, 'kind', NEW.kind, 'state', NEW.state,
    'prNumber', NEW.pr_number, 'prUrl', NEW.pr_url));
  RETURN NULL;
END
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
CREATE TRIGGER stream_revision AFTER INSERT OR UPDATE ON revision
FOR EACH ROW EXECUTE FUNCTION crosswatc_stream_revision();

-- +goose StatementBegin
CREATE FUNCTION crosswatc_stream_operation() RETURNS trigger AS $$
BEGIN
  IF TG_OP = 'UPDATE' AND to_jsonb(OLD) = to_jsonb(NEW) THEN
    RETURN NULL;
  END IF;
  PERFORM crosswatc_stream_publish('operation.updated', NEW.resource_id, jsonb_build_object(
    'resourceId', NEW.resource_id, 'operationId', NEW.id, 'revisionId', NEW.revision_id, 'type', NEW.type,
    'status', NEW.status));
  RETURN NULL;
END
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
CREATE TRIGGER stream_operation AFTER INSERT OR UPDATE ON operation
FOR EACH ROW EXECUTE FUNCTION crosswatc_stream_operation();

-- +goose StatementBegin
CREATE FUNCTION crosswatc_stream_event() RETURNS trigger AS $$
BEGIN
  PERFORM crosswatc_stream_publish('event.created', NEW.resource_id, jsonb_build_object(
    'resourceId', NEW.resource_id, 'eventId', NEW.id, 'revisionId', NEW.revision_id, 'source', NEW.source,
    'type', NEW.type, 'level', NEW.level, 'message', NEW.message, 'createdAt', NEW.created_at));
  RETURN NULL;
END
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
CREATE TRIGGER stream_tracker_event AFTER INSERT ON tracker_event
FOR EACH ROW EXECUTE FUNCTION crosswatc_stream_event();

-- +goose StatementBegin
CREATE FUNCTION crosswatc_stream_topology() RETURNS trigger AS $$
BEGIN
  IF TG_OP = 'UPDATE' AND OLD.nodes = NEW.nodes AND OLD.edges = NEW.edges THEN
    RETURN NULL;
  END IF;
  PERFORM crosswatc_stream_publish('topology.updated', NEW.resource_id, jsonb_build_object('resourceId', NEW.resource_id));
  RETURN NULL;
END
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
CREATE TRIGGER stream_topology AFTER INSERT OR UPDATE ON resource_topology
FOR EACH ROW EXECUTE FUNCTION crosswatc_stream_topology();

-- Projection: lifecycle stages/health changes refresh the resource; drift turning on emits drift.detected.
-- +goose StatementBegin
CREATE FUNCTION crosswatc_stream_projection() RETURNS trigger AS $$
BEGIN
  IF TG_OP = 'INSERT' OR OLD.stages IS DISTINCT FROM NEW.stages OR OLD.health IS DISTINCT FROM NEW.health THEN
    PERFORM crosswatc_stream_publish('resource.updated', NEW.resource_id, jsonb_build_object(
      'resourceId', NEW.resource_id, 'reason', 'projection'));
  END IF;
  IF coalesce(NEW.drift_state->>'detected', 'false') = 'true'
     AND (TG_OP = 'INSERT' OR coalesce(OLD.drift_state->>'detected', 'false') <> 'true') THEN
    PERFORM crosswatc_stream_publish('drift.detected', NEW.resource_id, jsonb_build_object(
      'resourceId', NEW.resource_id, 'drift', NEW.drift_state));
  END IF;
  RETURN NULL;
END
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
CREATE TRIGGER stream_projection AFTER INSERT OR UPDATE ON resource_projection
FOR EACH ROW EXECUTE FUNCTION crosswatc_stream_projection();

GRANT SELECT, INSERT, DELETE ON stream_event TO crosswatc_app;
GRANT USAGE ON SEQUENCE stream_event_id_seq TO crosswatc_app;

-- +goose Down
DROP TRIGGER stream_projection ON resource_projection;
DROP TRIGGER stream_topology ON resource_topology;
DROP TRIGGER stream_tracker_event ON tracker_event;
DROP TRIGGER stream_operation ON operation;
DROP TRIGGER stream_revision ON revision;
DROP TRIGGER stream_resource ON tracked_resource;
DROP FUNCTION crosswatc_stream_projection(), crosswatc_stream_topology(), crosswatc_stream_event(),
  crosswatc_stream_operation(), crosswatc_stream_revision(), crosswatc_stream_resource(),
  crosswatc_stream_publish(TEXT, UUID, JSONB);
DROP TABLE stream_event;
