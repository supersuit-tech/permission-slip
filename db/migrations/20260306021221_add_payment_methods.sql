-- +goose Up

-- Stored payment methods for agent-initiated purchases.
-- Raw card data is never stored — only Stripe payment method tokens and
-- display metadata (last4, brand, expiry). Users set optional spending limits
-- (per-transaction and monthly) to control agent spend.
CREATE TABLE payment_methods (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    stripe_payment_method_id TEXT NOT NULL UNIQUE,
    label                   TEXT NOT NULL DEFAULT '',
    brand                   TEXT NOT NULL DEFAULT '',
    last4                   TEXT NOT NULL DEFAULT '',
    exp_month               INT NOT NULL DEFAULT 0,
    exp_year                INT NOT NULL DEFAULT 0,
    billing_address_line1   TEXT NOT NULL DEFAULT '',
    billing_address_line2   TEXT NOT NULL DEFAULT '',
    billing_address_city    TEXT NOT NULL DEFAULT '',
    billing_address_state   TEXT NOT NULL DEFAULT '',
    billing_address_postal  TEXT NOT NULL DEFAULT '',
    billing_address_country TEXT NOT NULL DEFAULT '',
    is_default              BOOLEAN NOT NULL DEFAULT false,
    per_transaction_limit   INT,       -- cents; NULL = no limit
    monthly_limit           INT,       -- cents; NULL = no limit
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_payment_methods_user_id ON payment_methods(user_id);

-- Ensure at most one default payment method per user.
CREATE UNIQUE INDEX idx_payment_methods_user_default
    ON payment_methods(user_id) WHERE is_default = true;

-- Enable RLS — access control is enforced in application code via
-- WHERE user_id = $1 clauses. RLS policies (using auth.uid()) are
-- configured at the Supabase level, not in migrations.
ALTER TABLE payment_methods ENABLE ROW LEVEL SECURITY;

-- Transaction log for monthly spending limit enforcement.
-- Records each charge against a stored payment method so we can sum
-- the rolling monthly total before allowing a new charge.
CREATE TABLE payment_method_transactions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_method_id UUID NOT NULL REFERENCES payment_methods(id) ON DELETE CASCADE,
    user_id           UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    connector_id      TEXT NOT NULL DEFAULT '',
    action_type       TEXT NOT NULL DEFAULT '',
    amount_cents      INT NOT NULL,
    description       TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_pmt_payment_method_id ON payment_method_transactions(payment_method_id);
CREATE INDEX idx_pmt_user_created ON payment_method_transactions(user_id, created_at);

ALTER TABLE payment_method_transactions ENABLE ROW LEVEL SECURITY;

-- +goose Down

DROP TABLE IF EXISTS payment_method_transactions;
DROP TABLE IF EXISTS payment_methods;
