-- +goose Up

-- Non-secret connector sync metadata (e.g. Proton Mail UIDVALIDITY per folder).
-- Secrets remain in the vault; this column is updated during connector execution.
ALTER TABLE credentials ADD COLUMN connector_state jsonb NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE credentials DROP COLUMN IF EXISTS connector_state;
