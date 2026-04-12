-- +goose Up

ALTER TABLE action_config_templates
    ADD COLUMN standing_approval_spec jsonb
        CHECK (standing_approval_spec IS NULL OR jsonb_typeof(standing_approval_spec) = 'object')
        CHECK (standing_approval_spec IS NULL OR pg_column_size(standing_approval_spec) <= 4096);

-- +goose Down

ALTER TABLE action_config_templates
    DROP COLUMN IF EXISTS standing_approval_spec;
