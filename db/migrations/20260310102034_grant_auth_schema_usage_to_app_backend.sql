-- +goose Up

-- The app_backend role was granted SELECT on auth.users (in the cascade
-- migration) but was never granted USAGE on the auth schema. In PostgreSQL,
-- both USAGE on the schema and the table-level privilege are required.
-- Without schema USAGE, every query touching auth.users (FindProfileByAuthEmail,
-- RelinkProfile, CreateProfile) fails with 42501 insufficient_privilege —
-- making profile re-linking and new-user onboarding silently fail in
-- production.
GRANT USAGE ON SCHEMA auth TO app_backend;

-- +goose Down
REVOKE USAGE ON SCHEMA auth FROM app_backend;
