-- +goose Up
-- Per-user, per-channel notification preferences. Missing rows default to
-- enabled — insert a row with enabled=false to opt out of a channel.

CREATE TABLE notification_preferences (
    user_id uuid    NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    channel text    NOT NULL CHECK (channel IN ('email', 'web-push', 'sms')),
    enabled boolean NOT NULL DEFAULT true,
    PRIMARY KEY (user_id, channel)
);

-- +goose Down
DROP TABLE IF EXISTS notification_preferences;
