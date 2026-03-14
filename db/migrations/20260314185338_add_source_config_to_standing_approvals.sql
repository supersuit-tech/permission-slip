-- +goose Up
-- Tracks which action configuration a standing approval was derived from.
-- Nullable because existing approvals predate this column and the source
-- config may be deleted after the standing approval is created (no FK).
ALTER TABLE standing_approvals
    ADD COLUMN source_action_configuration_id text;

-- +goose Down
ALTER TABLE standing_approvals
    DROP COLUMN IF EXISTS source_action_configuration_id;
