-- +goose Up
ALTER TABLE agents DROP CONSTRAINT agents_status_check;
ALTER TABLE agents ADD CONSTRAINT agents_status_check CHECK (status IN ('pending', 'registered', 'deactivated'));
ALTER TABLE agents ADD COLUMN deactivated_at timestamptz;

-- +goose Down
ALTER TABLE agents DROP COLUMN deactivated_at;
-- Remove any deactivated agents before restoring the original constraint
DELETE FROM agents WHERE status = 'deactivated';
ALTER TABLE agents DROP CONSTRAINT agents_status_check;
ALTER TABLE agents ADD CONSTRAINT agents_status_check CHECK (status IN ('pending', 'registered'));
