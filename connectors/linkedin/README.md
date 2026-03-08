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
| `r_organization_social` | Reading organization post analytics; basic org profile lookup |
| `w_organization_social` | Creating posts on behalf of company pages |

> **Partner-only scopes:** `w_messages` (send_message) and `r_network` (list_connections) require
> LinkedIn Partner Program or product-specific approval. They are **not** included in the connector's
> standard OAuth scope list — adding them without partner approval breaks the OAuth flow entirely for
> standard apps. Apps with partner approval must request these scopes out-of-band.

## Actions

### Standard Access

| Action Type | Name | Risk | Description |
|---|---|---|---|
| `linkedin.create_post` | Create Post | medium | Create a text or link-share post on the user's feed |
| `linkedin.delete_post` | Delete Post | medium | Delete a post by URN (irreversible) |
| `linkedin.add_comment` | Add Comment | medium | Add a comment on any LinkedIn post |
| `linkedin.get_profile` | Get My Profile | low | Get the authenticated user's profile info |
| `linkedin.get_post_analytics` | Get Post Analytics | low | Get likes/comments metrics for a post |
| `linkedin.create_company_post` | Create Company Post | high | Post on behalf of a company page |
| `linkedin.get_company` | Get Company | low | Look up a LinkedIn company profile by organization ID |

### Marketing Developer Platform (MDP) Access Required

These actions require MDP or Sales Navigator API access. Standard OAuth apps receive HTTP 403.

| Action Type | Name | Risk | Description |
|---|---|---|---|
| `linkedin.search_people` | Search People | low | Search LinkedIn members by keywords, company, and/or title |
| `linkedin.search_companies` | Search Companies | low | Search LinkedIn company pages by keyword |

### LinkedIn Partner Program Approval Required

These actions require additional LinkedIn partner privileges beyond standard developer apps.

| Action Type | Name | Risk | Description | Extra Scope |
|---|---|---|---|---|
| `linkedin.send_message` | Send Message | high | Send a direct message to a connection | `w_messages` |
| `linkedin.list_connections` | List Connections | low | List the authenticated user's 1st-degree connections | `r_network` |

### Post Text Length

Post text (`commentary`) is capped at 3,000 characters. Comment text is capped at 1,250 characters. Exceeding these limits returns a `ValidationError` before hitting the LinkedIn API.

### Post Visibility

- **`PUBLIC`** — Visible to anyone on LinkedIn (default for personal and company posts)
- **`CONNECTIONS`** — Visible only to the user's connections (personal posts only; company posts are always `PUBLIC`)

### Post URN

When creating a post, the LinkedIn API returns the post URN in the `x-restli-id` response header. This URN (e.g., `urn:li:share:7654321`) is included in the result as `post_urn` and can be used to delete the post or retrieve its analytics.

### Link Shares

Both `create_post` and `create_company_post` support sharing links by providing `article_url`. The URL is validated before sending to the API. Optional `article_title` and `article_description` fields control the link preview.

### Message Limits

Direct messages (`send_message`) have a body limit of 8,000 characters and an optional subject limit of 200 characters. These are validated before the API call. Only `urn:li:person:{numeric_id}` URNs are accepted as recipients — share URNs and non-person URNs are rejected.

### Pagination

`search_people`, `search_companies`, and `list_connections` all return a `next_start` field that equals `start + len(results)`. Pass it as the `start` parameter on the next request to page through results.

- Search actions default to 10 results per page with a maximum of 50.
- `list_connections` defaults to 20 results per page with a maximum of 500.

### Localized Strings

Company profile fields (name, description) are returned in the preferred locale of the organization. When the preferred locale key is not present, the connector falls back to the lexicographically first available key to ensure a deterministic result.

### URN Formats

LinkedIn uses URNs to identify resources:

| Type | Format | Used In |
|------|--------|---------|
| Person | `urn:li:person:{numeric_id}` | recipient_urn in send_message |
| Organization | `urn:li:organization:{numeric_id}` | company profiles, organization_urn fields |
| Share/Post | `urn:li:share:{numeric_id}` | post_urn in create_post results |

The connector validates URN format before making API calls to prevent malformed requests and to ensure person URNs are not confused with other URN types.

## Error Handling

| HTTP Status | Connector Error | Description |
|---|---|---|
| 401 | `AuthError` | Invalid or expired access token |
| 403 | `AuthError` | Insufficient permissions (missing OAuth scopes or partner access) |
| 422 | `ValidationError` | LinkedIn rejected the request payload |
| 429 | `RateLimitError` | Rate limit exceeded; respects `Retry-After` header |
| 5xx | `ExternalError` | LinkedIn API server error |

LinkedIn error responses include a `serviceErrorCode` for more specific diagnostics, which is appended to error messages when available.

## File Structure

```
connectors/linkedin/
├── linkedin.go                  # Connector struct, HTTP client, error handling, validation helpers
├── linkedin_test.go             # Connector-level tests (ID, actions, credentials, manifest)
├── manifest.go                  # Manifest with actions, JSON schemas, templates
├── register.go                  # Registers the connector with the platform
├── pagination.go                # Shared pagination types (searchPaging), constants, and helpers
│                                # (validateCountStart, resolveCount, nextStart)
├── helpers_test.go              # Shared test helpers (newForTest, validCreds)
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
├── send_message.go              # linkedin.send_message action
├── send_message_test.go
├── search_people.go             # linkedin.search_people action
├── search_people_test.go
├── search_companies.go          # linkedin.search_companies action
├── search_companies_test.go
├── get_company.go               # linkedin.get_company action + localizedString helpers
├── get_company_test.go
├── list_connections.go          # linkedin.list_connections action
├── list_connections_test.go
└── README.md                    # This file
```

## Testing

```bash
go test ./connectors/linkedin/... -v
```

All tests use `httptest.NewServer` to mock the LinkedIn API — no real API calls are made.

### Test Coverage Notes

- **Access-tier gating** (MDP/Partner) is tested at the validation layer only — tests verify that
  parameter validation errors are returned correctly. End-to-end behavior with real LinkedIn
  credentials must be verified manually with an account that has the required access tier.
- **Pagination** tests verify `next_start` computation and boundary validation (negative start,
  count exceeds max).
- **URN validation** tests confirm that only `urn:li:person:{numeric_id}` is accepted for messaging
  — share URNs, non-person URNs, and freeform strings are all rejected.
