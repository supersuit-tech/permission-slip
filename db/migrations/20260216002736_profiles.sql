-- +goose Up

-- Supabase provides the auth schema and auth.users table.
-- Create a minimal version only when running against standalone Postgres
-- (e.g. CI/tests). Skip when Supabase already owns the auth schema.
-- +goose StatementBegin
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'auth') THEN
        CREATE SCHEMA auth;
        CREATE TABLE auth.users (id uuid PRIMARY KEY);
    END IF;
END $$;
-- +goose StatementEnd

-- Enable pg_cron for scheduled cleanup jobs (available on Supabase;
-- gracefully skipped in local development where the extension isn't installed).
-- +goose StatementBegin
DO $$ BEGIN
    CREATE EXTENSION IF NOT EXISTS "pg_cron";
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'pg_cron not available (%), skipping', SQLERRM;
END $$;
-- +goose StatementEnd

-- Helper functions for pg_cron operations. These wrap cron.schedule() and
-- cron.unschedule() with graceful error handling so migrations work in
-- environments where pg_cron is not installed (e.g. local development).

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION try_cron_schedule(job_name text, schedule text, command text)
RETURNS void LANGUAGE plpgsql AS $$
BEGIN
    PERFORM cron.schedule(job_name, schedule, command);
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'pg_cron not available (%), skipping schedule of %', SQLERRM, job_name;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION try_cron_unschedule(job_name text)
RETURNS void LANGUAGE plpgsql AS $$
BEGIN
    PERFORM cron.unschedule(job_name);
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'pg_cron not available (%), skipping unschedule of %', SQLERRM, job_name;
END;
$$;
-- +goose StatementEnd

CREATE TABLE profiles (
    id         uuid        PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
    username   text        UNIQUE NOT NULL CHECK (char_length(username) <= 255),
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS profiles;
DROP FUNCTION IF EXISTS try_cron_unschedule(text);
DROP FUNCTION IF EXISTS try_cron_schedule(text, text, text);
