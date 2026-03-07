# X (Twitter) Connector

The X connector integrates Permission Slip with the [X API v2](https://developer.x.com/en/docs/x-api). It uses plain `net/http` — no third-party SDK.

## Connector ID

`x`

## Authentication

The connector uses OAuth 2.0 with PKCE via the X OAuth provider. Tokens are stored encrypted in Supabase Vault and decrypted only at execution time.

| Key | Required | Description |
|-----|----------|-------------|
| `access_token` | Yes | An OAuth 2.0 Bearer token with appropriate scopes. |

**Required OAuth scopes:** `tweet.read`, `tweet.write`, `users.read`, `dm.read`, `dm.write`, `offline.access`

**OAuth endpoints:**
- Authorize: `https://x.com/i/oauth2/authorize`
- Token: `https://api.x.com/2/oauth2/token`

## Actions

### `x.post_tweet`

Post a tweet, reply, or quote tweet.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `text` | string | Yes | Tweet text (max 280 characters) |
| `reply_to_tweet_id` | string | No | Tweet ID to reply to |
| `quote_tweet_id` | string | No | Tweet ID to quote |
| `media_ids` | string[] | No | Pre-uploaded media IDs to attach |

**X API:** `POST /2/tweets` ([docs](https://developer.x.com/en/docs/x-api/tweets/manage-tweets/api-reference/post-tweets))

---

### `x.delete_tweet`

Delete a tweet (irreversible).

**Risk level:** high

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `tweet_id` | string | Yes | ID of the tweet to delete |

**X API:** `DELETE /2/tweets/{id}` ([docs](https://developer.x.com/en/docs/x-api/tweets/manage-tweets/api-reference/delete-tweets-id))

---

### `x.send_dm`

Send a direct message to a user.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `recipient_id` | string | Yes | User ID of the recipient |
| `text` | string | Yes | Message text (max 10,000 characters) |

**X API:** `POST /2/dm_conversations/with/{participant_id}/messages` ([docs](https://developer.x.com/en/docs/x-api/direct-messages/manage/api-reference/post-dm-conversation-with-user))

---

### `x.get_user_tweets`

Get recent tweets from a specific user.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `user_id` | string | Yes | — | User ID to get tweets from |
| `max_results` | integer | No | 10 | Maximum results (1-100) |
| `since_id` | string | No | — | Only return tweets after this tweet ID |
| `until_id` | string | No | — | Only return tweets before this tweet ID |

**X API:** `GET /2/users/{id}/tweets` ([docs](https://developer.x.com/en/docs/x-api/tweets/timelines/api-reference/get-users-id-tweets))

---

### `x.search_tweets`

Search recent tweets (7-day window on Basic tier).

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `query` | string | Yes | — | Search query ([X search syntax](https://developer.x.com/en/docs/x-api/tweets/search/integrate/build-a-query)) |
| `max_results` | integer | No | 10 | Maximum results (10-100) |
| `since_id` | string | No | — | Only return tweets after this tweet ID |
| `sort_order` | string | No | — | `recency` or `relevancy` |

**X API:** `GET /2/tweets/search/recent` ([docs](https://developer.x.com/en/docs/x-api/tweets/search/api-reference/get-tweets-search-recent))

---

### `x.get_me`

Get the authenticated user's profile info.

**Risk level:** low

**Parameters:** None

**X API:** `GET /2/users/me` ([docs](https://developer.x.com/en/docs/x-api/users/lookup/api-reference/get-users-me))

## Error Handling

The connector maps X API error responses to typed connector errors:

| HTTP Status | Connector Error | Description |
|-------------|-----------------|-------------|
| 401 | `AuthError` | Invalid or expired token |
| 403 | `AuthError` | Insufficient permissions |
| 429 | `RateLimitError` | Rate limit exceeded (includes retry-after) |
| Other 4xx/5xx | `ExternalError` | X API error with message |
| Client timeout | `TimeoutError` | Request timed out (30s default) |

Rate limit responses use the `x-rate-limit-reset` header to determine retry timing (defaults to 30s if missing).

Response body reads are capped at 10 MB to prevent OOM from unexpectedly large responses.

## Adding a New Action

Each action lives in its own file. To add one (e.g., `x.like_tweet`):

1. Create `connectors/x/like_tweet.go` with a params struct, `validate()`, and an `Execute` method.
2. Use `a.conn.do(ctx, creds, method, path, body, &resp)` for the HTTP lifecycle — it handles Bearer auth, rate limiting, timeouts, and response size limits.
3. Return `connectors.JSONResult(respBody)` to wrap the response.
4. Register the action in `Actions()` inside `x.go`.
5. Add the action to the manifest in `manifest.go`.
6. Add tests in `like_tweet_test.go` using `httptest.NewServer` and `newForTest()`.

## File Structure

```
connectors/x/
├── x.go                    # XConnector struct, New(), Actions(), do(), checkResponse()
├── manifest.go             # Manifest() — actions, templates, OAuth provider, credentials
├── post_tweet.go           # x.post_tweet action
├── delete_tweet.go         # x.delete_tweet action
├── send_dm.go              # x.send_dm action
├── get_user_tweets.go      # x.get_user_tweets action
├── search_tweets.go        # x.search_tweets action
├── get_me.go               # x.get_me action
├── x_test.go               # Connector-level tests
├── helpers_test.go         # Shared test helpers (validCreds)
├── *_test.go               # Per-action tests
└── README.md               # This file
```

## Testing

All tests use `httptest.NewServer` to mock the X API — no real API calls are made.

```bash
go test ./connectors/x/... -v
```
