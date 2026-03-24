# Netlify Connector

The Netlify connector integrates Permission Slip with the [Netlify API](https://docs.netlify.com/api/). It uses plain `net/http` — no third-party SDK.

## Connector ID

`netlify`

## Authentication

The Netlify connector supports two authentication methods. OAuth is recommended and presented as the default in the UI.

### OAuth (Recommended)

| Service | Auth Type | Provider |
|---------|-----------|----------|
| `netlify` | `oauth2` | `netlify` |

OAuth tokens are automatically refreshed. Users connect via the OAuth flow in the connector settings. Requires `NETLIFY_CLIENT_ID` and `NETLIFY_CLIENT_SECRET` environment variables.

Netlify does not use granular OAuth scopes — tokens receive full account access.

### API Key (Alternative)

| Service | Auth Type | Description |
|---------|-----------|-------------|
| `netlify-api-key` | `api_key` | A Netlify personal access token. Create one at **User Settings → Applications → Personal access tokens**. |

At execution time, the system tries OAuth first. If the user has no OAuth connection, it falls back to a stored API key. All tokens are encrypted in Supabase Vault and decrypted only at execution time.

## Actions

### `netlify.list_sites`

List all sites in the account.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `filter` | string | No | `"all"` | `"all"`, `"owner"`, or `"guest"` |
| `page` | integer | No | 1 | Page number |
| `per_page` | integer | No | 20 | Results per page (max 100) |

**Netlify API:** `GET /sites`

---

### `netlify.list_deployments`

List deployments for a specific site.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `site_id` | string | Yes | Site ID or subdomain |
| `page` | integer | No | Page number |
| `per_page` | integer | No | Results per page (max 100) |

**Netlify API:** `GET /sites/{site_id}/deploys`

---

### `netlify.get_deployment`

Get detailed status and information for a specific deployment.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `deploy_id` | string | Yes | The deploy ID |

**Netlify API:** `GET /deploys/{deploy_id}`

---

### `netlify.trigger_deployment`

Trigger a new build and deployment for a site.

**Risk level:** high

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `site_id` | string | Yes | — | Site ID or subdomain |
| `branch` | string | No | production branch | Git branch to deploy |
| `clear_cache` | boolean | No | `false` | Clear build cache before deploying |
| `title` | string | No | — | Optional deploy title for identification |

**Netlify API:** `POST /sites/{site_id}/builds`

---

### `netlify.rollback_deployment`

Rollback a site to a previous deployment by publishing it.

**Risk level:** high

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `site_id` | string | Yes | Site ID or subdomain |
| `deploy_id` | string | Yes | Deploy ID to restore |

**Netlify API:** `POST /sites/{site_id}/deploys/{deploy_id}/restore`

---

### `netlify.list_env_vars`

List all environment variables for a site.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_slug` | string | Yes | Account slug (team name) |
| `site_id` | string | Yes | Site ID to scope the listing |

**Netlify API:** `GET /accounts/{account_slug}/env?site_id={site_id}`

---

### `netlify.set_env_var`

Create or update an environment variable for a site.

**Risk level:** high

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_slug` | string | Yes | Account slug (team name) |
| `site_id` | string | Yes | Site ID to scope the variable |
| `key` | string | Yes | Variable name |
| `values` | array | Yes | Values per deploy context (see below) |

Each entry in `values` has:
- `value` (string) — the variable value
- `context` (string) — one of `"all"`, `"dev"`, `"branch-deploy"`, `"deploy-preview"`, `"production"`

**Netlify API:** `POST /accounts/{account_slug}/env?site_id={site_id}`

---

### `netlify.delete_env_var`

Delete an environment variable from a site.

**Risk level:** high

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_slug` | string | Yes | Account slug (team name) |
| `key` | string | Yes | Variable name to delete |
| `site_id` | string | No | Site ID to scope the deletion |

**Netlify API:** `DELETE /accounts/{account_slug}/env/{key}`

## Error Handling

| Netlify Status | Connector Error | HTTP Response |
|----------------|-----------------|---------------|
| 401, 403 | `AuthError` | 502 Bad Gateway |
| 400, 422 | `ValidationError` | 400 Bad Request |
| 404 | `ValidationError` | 400 Bad Request |
| 429 | `RateLimitError` | 429 Too Many Requests |
| Other 4xx/5xx | `ExternalError` | 502 Bad Gateway |
| Client timeout | `TimeoutError` | 504 Gateway Timeout |

## Recommended Agent Workflows

**Deploy and monitor:**
1. `netlify.trigger_deployment` to start a build
2. `netlify.list_deployments` or `netlify.get_deployment` to check build progress
3. On failure, `netlify.rollback_deployment` to restore the previous version

**Environment variable management:**
1. `netlify.list_env_vars` to see current configuration
2. `netlify.set_env_var` with context-specific values (e.g., different DB URLs for preview vs production)

## File Structure

```
connectors/netlify/
├── netlify.go               # Connector struct, Manifest(), Actions(), do()
├── list_sites.go            # netlify.list_sites
├── list_deployments.go      # netlify.list_deployments
├── get_deployment.go        # netlify.get_deployment
├── trigger_deployment.go    # netlify.trigger_deployment
├── rollback_deployment.go   # netlify.rollback_deployment
├── env_vars.go              # netlify.list_env_vars, set_env_var, delete_env_var
├── response.go              # HTTP response → typed error mapping
├── netlify_test.go          # Connector-level tests
├── actions_test.go          # Action-level tests
├── helpers_test.go          # Shared test helpers
└── README.md                # This file
```
