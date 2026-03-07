# Vercel Connector

The Vercel connector integrates Permission Slip with the [Vercel REST API](https://vercel.com/docs/rest-api). It uses plain `net/http` — no third-party SDK.

## Connector ID

`vercel`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `api_key` | Yes | A Vercel API token. Create one at **Account Settings → Tokens**. For team projects, the token must belong to a team member with appropriate permissions. |

The credential `auth_type` in the database is `api_key`. Tokens are stored encrypted in Supabase Vault and decrypted only at execution time.

## Team vs Personal Accounts

Most actions accept an optional `team_id` parameter. For **personal accounts**, omit it. For **team accounts**, include the team ID (found in **Team Settings → General**) to scope requests to that team.

## Actions

### `vercel.list_projects`

List all projects in the account.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `team_id` | string | No | Team ID (required for team accounts) |
| `limit` | integer | No | Max results (default 20, max 100) |

---

### `vercel.list_deployments`

List deployments with optional filtering.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `project_id` | string | No | Filter by project ID or name |
| `team_id` | string | No | Team ID |
| `target` | string | No | `"production"` or `"preview"` |
| `state` | string | No | `"BUILDING"`, `"ERROR"`, `"INITIALIZING"`, `"QUEUED"`, `"READY"`, `"CANCELED"` |
| `limit` | integer | No | Max results (default 20, max 100) |

---

### `vercel.get_deployment`

Get detailed status and information for a specific deployment.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `deployment_id` | string | Yes | The deployment ID or URL |
| `team_id` | string | No | Team ID |

**Vercel API:** `GET /v13/deployments/{id}`

---

### `vercel.trigger_deployment`

Create a new deployment from a Git branch or commit.

**Risk level:** high

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `project_id` | string | Yes | — | Project ID or name |
| `ref` | string | Yes | — | Git ref (branch, tag, or commit SHA) |
| `ref_type` | string | No | `"branch"` | Type of Git ref: `"branch"`, `"commit"`, or `"tag"` |
| `target` | string | No | `"preview"` | `"production"` or `"preview"` |
| `team_id` | string | No | — | Team ID |

**Vercel API:** `POST /v13/deployments`

---

### `vercel.promote_deployment`

Promote an existing preview deployment to production. This is the recommended safe workflow: deploy a preview, verify it works, then promote.

**Risk level:** high

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `project_id` | string | Yes | Project ID or name |
| `deployment_id` | string | Yes | Preview deployment ID to promote |
| `team_id` | string | No | Team ID |

**Vercel API:** `POST /v10/projects/{id}/promote`

---

### `vercel.rollback_deployment`

Rollback a project to a previous deployment.

**Risk level:** high

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `project_id` | string | Yes | Project ID or name |
| `deployment_id` | string | Yes | Deployment ID to rollback to |
| `team_id` | string | No | Team ID |

**Vercel API:** `POST /v9/projects/{id}/rollback`

---

### `vercel.list_env_vars`

List all environment variables for a project.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `project_id` | string | Yes | Project ID or name |
| `team_id` | string | No | Team ID |

**Vercel API:** `GET /v9/projects/{id}/env`

---

### `vercel.set_env_var`

Create or update an environment variable.

**Risk level:** high

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `project_id` | string | Yes | — | Project ID or name |
| `key` | string | Yes | — | Variable name |
| `value` | string | Yes | — | Variable value |
| `target` | array | Yes | — | Environments: `"production"`, `"preview"`, `"development"` |
| `type` | string | No | `"encrypted"` | `"system"`, `"secret"`, `"encrypted"`, `"plain"`, `"sensitive"` |
| `team_id` | string | No | — | Team ID |

**Vercel API:** `POST /v10/projects/{id}/env`

---

### `vercel.delete_env_var`

Delete an environment variable from a project.

**Risk level:** high

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `project_id` | string | Yes | Project ID or name |
| `env_id` | string | Yes | Environment variable ID |
| `team_id` | string | No | Team ID |

**Vercel API:** `DELETE /v9/projects/{id}/env/{envId}`

## Error Handling

| Vercel Status | Connector Error | HTTP Response |
|---------------|-----------------|---------------|
| 401, 403 | `AuthError` | 502 Bad Gateway |
| 400, 422 | `ValidationError` | 400 Bad Request |
| 429 | `RateLimitError` | 429 Too Many Requests |
| Other 4xx/5xx | `ExternalError` | 502 Bad Gateway |
| Client timeout | `TimeoutError` | 504 Gateway Timeout |

## Recommended Agent Workflows

**Preview-then-promote (safest):**
1. `vercel.trigger_deployment` with `target: "preview"` (auto-approved via template)
2. `vercel.get_deployment` to check build status
3. `vercel.promote_deployment` to ship to production (requires approval)

**Monitor deployments:**
1. `vercel.list_deployments` to see recent activity
2. `vercel.get_deployment` for detailed status on a specific deploy

## File Structure

```
connectors/vercel/
├── vercel.go                # Connector struct, Manifest(), Actions(), do()
├── list_projects.go         # vercel.list_projects
├── list_deployments.go      # vercel.list_deployments
├── get_deployment.go        # vercel.get_deployment
├── trigger_deployment.go    # vercel.trigger_deployment
├── promote_deployment.go    # vercel.promote_deployment
├── rollback_deployment.go   # vercel.rollback_deployment
├── env_vars.go              # vercel.list_env_vars, set_env_var, delete_env_var
├── response.go              # HTTP response → typed error mapping
├── vercel_test.go           # Connector-level tests
├── actions_test.go          # Action-level tests
├── helpers_test.go          # Shared test helpers
└── README.md                # This file
```
