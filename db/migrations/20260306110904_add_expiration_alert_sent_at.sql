-- +goose Up

-- Track whether an expiration alert has already been sent for a payment method
-- so the daily cron doesn't send duplicates. Reset to NULL when the user updates
-- the card (e.g. new exp_month/exp_year from Stripe).
ALTER TABLE payment_methods ADD COLUMN expiration_alert_sent_at TIMESTAMPTZ;

-- +goose Down

ALTER TABLE payment_methods DROP COLUMN IF EXISTS expiration_alert_sent_at;
