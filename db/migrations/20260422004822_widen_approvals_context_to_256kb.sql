-- +goose Up
-- Allow larger approval context JSON after resource_details enrichment (Slack thread preview, etc.).
-- See issue #1004. action stays at 64 KiB; context raised to 256 KiB to match API + merge caps.

ALTER TABLE approvals DROP CONSTRAINT IF EXISTS approvals_context_check;

ALTER TABLE approvals ADD CONSTRAINT approvals_context_check
    CHECK (pg_column_size(context) <= 262144);

-- +goose Down

ALTER TABLE approvals DROP CONSTRAINT IF EXISTS approvals_context_check;

ALTER TABLE approvals ADD CONSTRAINT approvals_context_check
    CHECK (pg_column_size(context) <= 65536);
