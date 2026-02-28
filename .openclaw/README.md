# OpenClaw Configuration

This directory contains OpenClaw-specific configuration for automated PR verification.

## verify-setup.sh

Automated setup script run by the `verify-pr` skill before running tests.

**What it does:**
- Installs frontend dependencies if `node_modules` is missing or stale
- Downloads Go module dependencies
- Bootstraps `.env` from `.env.example` if not present
- Checks if PostgreSQL is installed and running
- Verifies the test database exists (creates if missing)

**When it runs:**
- Automatically during `openclaw verify <pr-url>` before tests
- Manually: `bash .openclaw/verify-setup.sh`

**Exit codes:**
- 0: Setup complete, ready for testing
- 1: Prerequisites missing or setup failed (see error message)

**Requirements:**
- PostgreSQL 16+ installed (`brew install postgresql@16`)
- PostgreSQL running (`brew services start postgresql@16`)
- `.env` file with DATABASE_URL_TEST (copied from `.env.example`)

See: https://github.com/openclaw/openclaw/blob/main/skills/verify-pr/references/setup-script-convention.md
