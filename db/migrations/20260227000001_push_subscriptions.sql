-- +goose Up

-- push_subscriptions stores Web Push subscriptions per user. A user may have
-- multiple subscriptions (one per browser/device). The endpoint URL is the
-- unique identifier provided by the browser's push service.
CREATE TABLE push_subscriptions (
    id         bigserial   PRIMARY KEY,
    user_id    uuid        NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    endpoint   text        NOT NULL,
    p256dh     text        NOT NULL,  -- base64url-encoded P-256 public key
    auth       text        NOT NULL,  -- base64url-encoded auth secret
    created_at timestamptz NOT NULL DEFAULT now(),

    -- Each browser endpoint is unique per user; re-subscribing replaces the old one.
    UNIQUE (user_id, endpoint)
);

CREATE INDEX idx_push_subscriptions_user_id ON push_subscriptions (user_id);

-- +goose Down
DROP TABLE IF EXISTS push_subscriptions;
