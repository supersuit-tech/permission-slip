# OPENCLAW.md — AI Agent Dev Guide

This file is for AI agents (like Openclaw/Chiedobot) working on Permission Slip locally.

## Local Dev Setup

- **Frontend:** http://localhost:5173 (Vite)
- **Backend API:** http://localhost:8080
- **Dev proxy:** http://localhost:3000 (routes `/supabase/*` → Supabase, everything else → Vite)

Use `localhost` URLs in the browser tool — ngrok shows a warning interstitial on first visit from Playwright that's hard to click through.

## Openclaw as a Registered Agent

When `TEST_AGENT_PUB_KEY` is set in the environment, the seed script registers an **Openclaw** agent for every test user that has agents. This lets you test the full agent flow locally with a real key pair.

### Setup

```bash
# Set your agent public key in ~/.zshrc
export TEST_AGENT_PUB_KEY="<your ssh public key>"

# Then reset the dev environment
bash ~/.openclaw/workspace/skills/reset-permission-slip/scripts/reset.sh
```

### Test Users (after seed)

| Email | Password | Has Openclaw agent? |
|-------|----------|---------------------|
| has-registered-agents@test.local | password123 | ✅ |
| has-recent-activity@test.local | password123 | ✅ |
| has-everything@test.local | password123 | ✅ |
| has-pending-approvals@test.local | password123 | ✅ |
| new@test.local | password123 | ❌ |
| has-access-requests@test.local | password123 | ❌ |

### Signing API Requests as an Agent

The Openclaw agent uses the private key at `~/.ssh/permission_slip_agent`.

Agent API requests must be signed — see the API docs or existing agent integration tests for the signing format.

## Workflow Defaults

- **Always open a PR** — don't commit to `main` directly (branch protection)
- **Branch naming:** `chiedobot/<short-description>`
- **DEV MODE only** — never use real credentials in local dev
