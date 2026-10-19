-- +goose Up
-- Outbox for user notifications (spec §60). Rows are created by a trigger on tracker_event, so events
-- inserted by the API or the worker notify alike; the worker delivers them with retries.
CREATE TABLE notification (
  id BIGSERIAL PRIMARY KEY,
  event_id UUID UNIQUE NOT NULL,
  resource_id UUID,
  owner TEXT,
  kind TEXT NOT NULL,              -- WAITING_APPROVAL | FAILED | OPERATIONAL | DRIFT_DETECTED
  payload JSONB NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending', -- pending | sent | failed | skipped
  attempts INTEGER NOT NULL DEFAULT 0,
  next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  sent_at TIMESTAMPTZ
);
CREATE INDEX notification_pending ON notification(next_attempt_at) WHERE status = 'pending';

-- +goose StatementBegin
CREATE FUNCTION crosswatc_enqueue_notification() RETURNS trigger AS $$
DECLARE
  v_kind TEXT;
  r tracked_resource%ROWTYPE;
  v_pr TEXT;
BEGIN
  v_kind := CASE NEW.type
    WHEN 'GITOPS.CHECK.SUCCEEDED' THEN 'WAITING_APPROVAL'
    WHEN 'INFRA.RECONCILE.FAILED' THEN 'FAILED'
    WHEN 'GITOPS.SYNC.FAILED' THEN 'FAILED'
    WHEN 'GITOPS.PR.FAILED' THEN 'FAILED'
    WHEN 'LIFECYCLE.OPERATIONAL' THEN 'OPERATIONAL'
    WHEN 'LIFECYCLE.DRIFT.DETECTED' THEN 'DRIFT_DETECTED'
  END;
  IF v_kind IS NULL OR NEW.resource_id IS NULL THEN
    RETURN NEW;
  END IF;
  SELECT * INTO r FROM tracked_resource WHERE id = NEW.resource_id;
  IF NEW.revision_id IS NOT NULL THEN
    SELECT pr_url INTO v_pr FROM revision WHERE id = NEW.revision_id;
  END IF;
  INSERT INTO notification (event_id, resource_id, owner, kind, payload)
  VALUES (NEW.id, NEW.resource_id, r.owner, v_kind, jsonb_build_object(
    'eventType', NEW.type, 'message', NEW.message, 'correlationId', NEW.correlation_id,
    'resourceName', r.name, 'environment', r.environment, 'kind', r.kind, 'provider', r.provider,
    'prUrl', v_pr, 'occurredAt', NEW.created_at))
  ON CONFLICT (event_id) DO NOTHING;
  RETURN NEW;
END
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER enqueue_notification AFTER INSERT ON tracker_event
FOR EACH ROW EXECUTE FUNCTION crosswatc_enqueue_notification();

GRANT SELECT, INSERT, UPDATE ON notification TO crosswatc_app;
GRANT USAGE ON SEQUENCE notification_id_seq TO crosswatc_app;

-- +goose Down
DROP TRIGGER enqueue_notification ON tracker_event;
DROP FUNCTION crosswatc_enqueue_notification();
DROP TABLE notification;
