-- +goose Up
-- Track consumed agent request signatures to prevent replay attacks.
--
-- Agent requests are authenticated with an Ed25519 signature over a canonical
-- string that includes the request method, path, query, timestamp, and body
-- hash. Without replay tracking, an attacker who intercepts a signed request
-- (stolen TLS logs, leaked HAR file, compromised proxy) can replay it any
-- number of times inside the 5-minute timestamp window.
--
-- Each successfully-verified signature is inserted here with an expiry equal
-- to `signed_timestamp + signatureTimestampWindow + skew_buffer`. Subsequent
-- attempts to consume the same signature collide on the primary key and are
-- rejected. Rows with expires_at in the past are no longer replayable (the
-- timestamp check already rejects them) and are reaped by a background job.
CREATE TABLE consumed_signatures (
    signature_hash bytea       PRIMARY KEY,           -- SHA-256(signature bytes)
    agent_id       bigint      NOT NULL,              -- for observability / partitioning; 0 for pre-registration
    expires_at     timestamptz NOT NULL,              -- when this row is safe to delete
    created_at     timestamptz NOT NULL DEFAULT now()
);

-- The cleanup job scans by expires_at — index accordingly.
CREATE INDEX idx_consumed_signatures_expires_at ON consumed_signatures (expires_at);

-- RLS + policy, matching the pattern in CLAUDE.md §app_backend Role Permissions.
ALTER TABLE consumed_signatures ENABLE ROW LEVEL SECURITY;
CREATE POLICY app_backend_all ON consumed_signatures FOR ALL TO app_backend USING (true) WITH CHECK (true);

-- pg_cron reaper (best-effort — safe to skip if pg_cron is unavailable).
-- The application also runs a Go cleanup worker; either is sufficient, but
-- running both avoids gaps when one or the other is disabled in a deployment.
SELECT try_cron_schedule(
    'cleanup_consumed_signatures',
    '*/5 * * * *',
    $$DELETE FROM consumed_signatures WHERE expires_at < now()$$
);

-- +goose Down
SELECT try_cron_unschedule('cleanup_consumed_signatures');
DROP TABLE IF EXISTS consumed_signatures;
