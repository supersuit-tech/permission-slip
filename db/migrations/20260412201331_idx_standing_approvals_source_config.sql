-- +goose Up

CREATE INDEX IF NOT EXISTS idx_standing_approvals_source_config_active
    ON standing_approvals (source_action_configuration_id)
    WHERE status = 'active';

-- +goose Down

DROP INDEX IF EXISTS idx_standing_approvals_source_config_active;
