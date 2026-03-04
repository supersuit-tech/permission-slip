-- +goose Up

-- Add 'mobile-push' to the allowed notification channels.
ALTER TABLE notification_preferences
    DROP CONSTRAINT IF EXISTS notification_preferences_channel_check;
ALTER TABLE notification_preferences
    ADD CONSTRAINT notification_preferences_channel_check
    CHECK (channel IN ('email', 'web-push', 'sms', 'mobile-push'));

-- +goose Down

ALTER TABLE notification_preferences
    DROP CONSTRAINT IF EXISTS notification_preferences_channel_check;
ALTER TABLE notification_preferences
    ADD CONSTRAINT notification_preferences_channel_check
    CHECK (channel IN ('email', 'web-push', 'sms'));
