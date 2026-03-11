# OPENCLAW.md — AI Agent Dev Guide

This file is for AI agents working on Permission Slip locally.

## Local Dev Setup

- **Frontend:** http://localhost:5173 (Vite)
- **Backend API:** http://localhost:8080
- **Dev proxy:** http://localhost:3000 (routes `/supabase/*` → Supabase, everything else → Vite)

> **Note for browser automation:** Use `localhost` URLs directly — ngrok shows a warning interstitial on first visit that can be difficult to click through programmatically.

## Registering Yourself as an Agent (Local Dev)

The seed script supports auto-registering an AI agent for every test user that has agents. This lets you test the full agent approval flow locally with a real key pair.

### Setup

1. Generate an SSH key pair (or use an existing one):
   ```bash
   ssh-keygen -t ed25519 -f ~/.ssh/my_agent_key -C "my-agent"
   ```

2. Export the public key as `TEST_AGENT_PUB_KEY` in your environment:
   ```bash
   # Add to ~/.zshrc or equivalent
   export TEST_AGENT_PUB_KEY="$(cat ~/.ssh/my_agent_key.pub)"
   ```

3. Reset the dev environment to trigger the seed:
   ```bash
   make dev-reset
   # or run the seed directly: go run ./cmd/seed
   ```

The seed will print instructions after completing, including which users your agent was registered for.

### Test Users (after seed)

| Email | Password | Gets agent when TEST_AGENT_PUB_KEY is set |
|-------|----------|-------------------------------------------|
| has-registered-agents@test.local | password123 | ✅ |
| has-recent-activity@test.local | password123 | ✅ |
| has-everything@test.local | password123 | ✅ |
| has-pending-approvals@test.local | password123 | ✅ |
| new@test.local | password123 | ❌ |
| has-access-requests@test.local | password123 | ❌ |

### Making API Calls as an Agent

Agent API requests must be signed with your private key. See the API docs or existing agent integration tests for the signing format.

Your private key path depends on where you generated it — the seed output will remind you which public key was registered.

## Workflow Defaults

- **Always open a PR** — `main` has branch protection, direct pushes are rejected
- **DEV MODE only** — never use real credentials in local dev
