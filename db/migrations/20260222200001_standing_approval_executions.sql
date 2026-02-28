-- +goose Up

CREATE TABLE standing_approval_executions (
    id              bigserial   PRIMARY KEY,
    standing_approval_id text   NOT NULL REFERENCES standing_approvals(standing_approval_id) ON DELETE CASCADE,
    parameters      jsonb       CHECK (pg_column_size(parameters) <= 65536),
    executed_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_sa_executions_sa_id ON standing_approval_executions (standing_approval_id);

-- +goose Down
DROP TABLE IF EXISTS standing_approval_executions;
