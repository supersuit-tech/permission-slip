-- +goose Up

ALTER TABLE oauth_connections ENABLE ROW LEVEL SECURITY;
ALTER TABLE oauth_provider_configs ENABLE ROW LEVEL SECURITY;

-- +goose Down

ALTER TABLE oauth_provider_configs DISABLE ROW LEVEL SECURITY;
ALTER TABLE oauth_connections DISABLE ROW LEVEL SECURITY;
