-- +goose Up

-- usage_periods: tracks per-user usage metrics per billing period.
-- Source of truth for request counting and billing.
--
-- Period boundaries follow the half-open interval convention: [period_start, period_end).
-- period_start is inclusive (first instant of the month), period_end is exclusive
-- (first instant of the NEXT month). For example, February 2026 is represented as
-- period_start=2026-02-01T00:00:00Z, period_end=2026-03-01T00:00:00Z.
CREATE TABLE usage_periods (
    id            bigserial   PRIMARY KEY,
    user_id       uuid        NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    period_start  timestamptz NOT NULL,
    period_end    timestamptz NOT NULL,
    request_count int         NOT NULL DEFAULT 0,
    sms_count     int         NOT NULL DEFAULT 0,
    created_at    timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT uq_usage_periods_user_period UNIQUE (user_id, period_start),
    CONSTRAINT chk_usage_periods_valid_range CHECK (period_end > period_start)
);

-- Note: The UNIQUE constraint on (user_id, period_start) creates a btree index
-- that Postgres can scan in reverse for ORDER BY period_start DESC queries.

-- +goose Down

DROP TABLE IF EXISTS usage_periods;
