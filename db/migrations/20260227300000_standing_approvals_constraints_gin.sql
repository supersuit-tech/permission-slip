-- +goose Up
-- GIN index on standing_approvals.constraints to optimize JSONB containment
-- queries (@>) for constraint matching at scale. Uses jsonb_path_ops for a
-- smaller, faster index tailored to containment checks. Partial index excludes
-- NULL constraints since they represent "no constraints" and don't need indexing.
CREATE INDEX IF NOT EXISTS idx_standing_approvals_constraints_gin
    ON standing_approvals USING gin (constraints jsonb_path_ops)
    WHERE constraints IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_standing_approvals_constraints_gin;
