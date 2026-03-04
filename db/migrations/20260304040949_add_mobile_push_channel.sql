-- +goose Up
-- Add mobile-push as a valid notification channel alongside email, web-push, and sms.
ALTER TABLE notification_preferences
    DROP CONSTRAINT notification_preferences_channel_check,
    ADD  CONSTRAINT notification_preferences_channel_check
         CHECK (channel IN ('email', 'web-push', 'sms', 'mobile-push'));

-- +goose Down
-- Revert to the original three channels. Any existing mobile-push rows must be
-- removed first to satisfy the restored constraint.
DELETE FROM notification_preferences WHERE channel = 'mobile-push';
ALTER TABLE notification_preferences
    DROP CONSTRAINT notification_preferences_channel_check,
    ADD  CONSTRAINT notification_preferences_channel_check
         CHECK (channel IN ('email', 'web-push', 'sms'));
