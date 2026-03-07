# LinkedIn Connector

The LinkedIn connector integrates Permission Slip with the [LinkedIn Marketing & Community Management APIs](https://learn.microsoft.com/en-us/linkedin/marketing/). It uses plain `net/http` with OAuth 2.0 access tokens — no third-party LinkedIn SDK.

## Connector ID

`linkedin`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `access_token` | Yes | OAuth 2.0 access token obtained via the LinkedIn OAuth flow. Managed automatically by the platform's OAuth infrastructure. |

The credential `auth_type` in the database is `oauth2` with provider `linkedin`. Tokens are stored encrypted in Supabase Vault and refreshed automatically by the platform.

### Authentication

All API requests use Bearer token authentication. The connector uses two API surfaces:

- **`/v2/userinfo`** — OpenID Connect userinfo endpoint (no version header needed)
- **`/rest/*`** — Versioned REST API endpoints with a required `LinkedIn-Version: 202401` header

### OAuth Scopes

| Scope | Required For |
|-------|-------------|
| `openid` | User identification (sub claim) |
| `profile` | Reading user profile information |
| `w_member_social` | Creating posts, comments on behalf of the user |
| `r_organization_social` | Reading organization post analytics |
| `w_organization_social` | Creating posts on behalf of company pages |

## Actions

| Action Type | Name | Risk | Description |
|---|---|---|---|
| `linkedin.create_post` | Create Post | medium | Create a text or link-share post on the user's feed |
| `linkedin.delete_post` | Delete Post | medium | Delete a post by URN (irreversible) |
| `linkedin.add_comment` | Add Comment | medium | Add a comment on any LinkedIn post |
| `linkedin.get_profile` | Get My Profile | low | Get the authenticated user's profile info |
| `linkedin.get_post_analytics` | Get Post Analytics | low | Get likes/comments metrics for a post |
| `linkedin.create_company_post` | Create Company Post | high | Post on behalf of a company page |

### Post Text Length

Post text (`commentary`) is capped at 3,000 characters. Comment text is capped at 1,250 characters. Exceeding these limits returns a `ValidationError` before hitting the LinkedIn API.

### Post Visibility

- **`PUBLIC`** — Visible to anyone on LinkedIn (default for personal and company posts)
- **`CONNECTIONS`** — Visible only to the user's connections (personal posts only; company posts are always `PUBLIC`)

### Post URN

When creating a post, the LinkedIn API returns the post URN in the `x-restli-id` response header. This URN (e.g., `urn:li:share:7654321`) is included in the result as `post_urn` and can be used to delete the post or retrieve its analytics.

### Link Shares

Both `create_post` and `create_company_post` support sharing links by providing `article_url`. The URL is validated before sending to the API. Optional `article_title` and `article_description` fields control the link preview.

## Error Handling

| HTTP Status | Connector Error | Description |
|---|---|---|
| 401 | `AuthError` | Invalid or expired access token |
| 403 | `AuthError` | Insufficient permissions (missing OAuth scopes) |
| 422 | `ValidationError` | LinkedIn rejected the request payload |
| 429 | `RateLimitError` | Rate limit exceeded; respects `Retry-After` header |
| 5xx | `ExternalError` | LinkedIn API server error |

LinkedIn error responses include a `serviceErrorCode` for more specific diagnostics, which is appended to error messages when available.

## File Structure

```
connectors/linkedin/
├── linkedin.go                  # Connector struct, HTTP client, error handling, validation helpers, getPersonURN
├── linkedin_test.go             # Connector-level tests (ID, actions, credentials, manifest)
├── manifest.go                  # Manifest with actions, schemas, templates
├── helpers_test.go              # Shared test credentials
├── validation_test.go           # Security validation tests (URN, org ID, URL scheme)
├── get_profile.go               # linkedin.get_profile action
├── get_profile_test.go
├── create_post.go               # linkedin.create_post action + shared request types
├── create_post_test.go
├── delete_post.go               # linkedin.delete_post action
├── delete_post_test.go
├── add_comment.go               # linkedin.add_comment action
├── add_comment_test.go
├── get_post_analytics.go        # linkedin.get_post_analytics action
├── get_post_analytics_test.go
├── create_company_post.go       # linkedin.create_company_post action
├── create_company_post_test.go
└── README.md                    # This file
```

## Testing

```bash
go test ./connectors/linkedin/... -v
```

All tests use `httptest.NewServer` to mock the LinkedIn API — no real API calls are made.
