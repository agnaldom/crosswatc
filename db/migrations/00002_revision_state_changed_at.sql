-- +goose Up
ALTER TABLE revision ADD COLUMN state_changed_at TIMESTAMPTZ;
UPDATE revision SET state_changed_at = coalesce(completed_at, started_at, now());
ALTER TABLE revision ALTER COLUMN state_changed_at SET NOT NULL;
ALTER TABLE revision ALTER COLUMN state_changed_at SET DEFAULT now();

-- +goose Down
ALTER TABLE revision DROP COLUMN state_changed_at;
