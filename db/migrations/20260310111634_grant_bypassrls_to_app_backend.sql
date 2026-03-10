-- +goose Up
-- The app_backend role needs BYPASSRLS because RLS is enabled on all
-- application tables with no permissive policies (RLS is intended to
-- block PostgREST/anon access only). Without BYPASSRLS, every query
-- from the Go backend returns zero rows.
ALTER ROLE app_backend BYPASSRLS;

-- +goose Down
ALTER ROLE app_backend NOBYPASSRLS;
