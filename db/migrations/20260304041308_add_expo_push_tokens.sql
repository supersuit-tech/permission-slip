-- +goose Up

-- expo_push_tokens stores Expo push tokens per user. A user may have
-- multiple tokens (one per mobile device). The token string is the unique
-- identifier provided by Expo's push notification service.
CREATE TABLE expo_push_tokens (
    id         bigserial   PRIMARY KEY,
    user_id    uuid        NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    token      text        NOT NULL,  -- Expo push token, e.g. "ExponentPushToken[...]"
    created_at timestamptz NOT NULL DEFAULT now(),

    -- Each Expo token is unique per user; re-registering replaces the old one.
    UNIQUE (user_id, token)
);

CREATE INDEX idx_expo_push_tokens_user_id ON expo_push_tokens (user_id);

-- Lock down PostgREST access (Go backend bypasses RLS as superuser).
ALTER TABLE expo_push_tokens ENABLE ROW LEVEL SECURITY;

-- +goose Down
ALTER TABLE expo_push_tokens DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS expo_push_tokens;
