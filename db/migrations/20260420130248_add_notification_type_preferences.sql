-- +goose Up
-- Per-user, per-notification-type preferences (e.g. silence auto-approval execution
-- notifications). Missing rows default to enabled — insert enabled=false to opt out.

CREATE TABLE notification_type_preferences (
    user_id uuid    NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    notification_type text NOT NULL CHECK (notification_type IN ('standing_execution')),
    enabled boolean NOT NULL DEFAULT true,
    PRIMARY KEY (user_id, notification_type)
);

ALTER TABLE notification_type_preferences ENABLE ROW LEVEL SECURITY;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_backend') THEN
        CREATE POLICY app_backend_all ON notification_type_preferences
            FOR ALL TO app_backend USING (true) WITH CHECK (true);
        GRANT SELECT, INSERT, UPDATE, DELETE ON notification_type_preferences TO app_backend;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS notification_type_preferences;
