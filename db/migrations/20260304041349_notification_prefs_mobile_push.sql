-- +goose Up
-- Add 'mobile-push' to the allowed notification channels.
ALTER TABLE notification_preferences
    DROP CONSTRAINT notification_preferences_channel_check;

ALTER TABLE notification_preferences
    ADD CONSTRAINT notification_preferences_channel_check
    CHECK (channel IN ('email', 'web-push', 'sms', 'mobile-push'));

-- +goose Down
-- Remove any mobile-push preferences before restoring the old constraint.
DELETE FROM notification_preferences WHERE channel = 'mobile-push';

ALTER TABLE notification_preferences
    DROP CONSTRAINT notification_preferences_channel_check;

ALTER TABLE notification_preferences
    ADD CONSTRAINT notification_preferences_channel_check
    CHECK (channel IN ('email', 'web-push', 'sms'));
