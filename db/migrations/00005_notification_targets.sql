-- +goose Up
-- Per-target delivery tracking: retries only resend to targets that have not received the message yet.
-- Status 'sending' marks rows leased by a worker (next_attempt_at is the lease expiry).
ALTER TABLE notification ADD COLUMN sent_targets TEXT[] NOT NULL DEFAULT '{}';
DROP INDEX notification_pending;
CREATE INDEX notification_due ON notification(next_attempt_at) WHERE status IN ('pending', 'sending');

-- +goose Down
DROP INDEX notification_due;
CREATE INDEX notification_pending ON notification(next_attempt_at) WHERE status = 'pending';
ALTER TABLE notification DROP COLUMN sent_targets;
