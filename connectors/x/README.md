# X (Twitter) Connector

The X connector integrates Permission Slip with the [X API v2](https://developer.x.com/en/docs/x-api). It uses plain `net/http` — no third-party SDK.

## Connector ID

`x`

## Authentication

The connector uses OAuth 2.0 with PKCE via the X OAuth provider. Tokens are stored encrypted in Supabase Vault and decrypted only at execution time.

| Key | Required | Description |
|-----|----------|-------------|
| `access_token` | Yes | An OAuth 2.0 Bearer token with appropriate scopes. |

**Required OAuth scopes:** `tweet.read`, `tweet.write`, `users.read`, `dm.read`, `dm.write`, `offline.access`, `like.read`, `like.write`, `follows.read`, `follows.write`

**OAuth endpoints:**
- Authorize: `https://x.com/i/oauth2/authorize`
- Token: `https://api.x.com/2/oauth2/token`

> **Note:** The engagement actions (`like_tweet`, `retweet`, `follow_user`, `get_followers`, `get_following`) require the additional scopes above. Existing tokens issued without those scopes will need to be re-authorized.

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
| `media_ids` | string[] | No | Pre-uploaded media IDs to attach (use `x.upload_media` first) |

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

---

### `x.like_tweet`

Like a tweet on behalf of the authenticated user.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `tweet_id` | string | Yes | ID of the tweet to like |
| `user_id` | string | No | Authenticated user's ID. If omitted, resolved automatically via `/users/me` |

**X API:** `POST /2/users/{id}/likes` ([docs](https://developer.x.com/en/docs/x-api/tweets/likes/api-reference/post-users-id-likes))

---

### `x.unlike_tweet`

Remove a like from a tweet.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `tweet_id` | string | Yes | ID of the tweet to unlike |
| `user_id` | string | No | Authenticated user's ID. If omitted, resolved automatically via `/users/me` |

**X API:** `DELETE /2/users/{id}/likes/{tweet_id}` ([docs](https://developer.x.com/en/docs/x-api/tweets/likes/api-reference/delete-users-id-likes-tweet_id))

---

### `x.retweet`

Retweet a tweet on behalf of the authenticated user.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `tweet_id` | string | Yes | ID of the tweet to retweet |
| `user_id` | string | No | Authenticated user's ID. If omitted, resolved automatically via `/users/me` |

**X API:** `POST /2/users/{id}/retweets` ([docs](https://developer.x.com/en/docs/x-api/tweets/retweets/api-reference/post-users-id-retweets))

---

### `x.unretweet`

Undo a retweet.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `tweet_id` | string | Yes | ID of the tweet to unretweet |
| `user_id` | string | No | Authenticated user's ID. If omitted, resolved automatically via `/users/me` |

**X API:** `DELETE /2/users/{id}/retweets/{source_tweet_id}` ([docs](https://developer.x.com/en/docs/x-api/tweets/retweets/api-reference/delete-users-id-retweets-source_tweet_id))

---

### `x.follow_user`

Follow a user on behalf of the authenticated user.

**Risk level:** medium

> **Note:** If the target account has protected tweets, `pending_follow` will be `true` in the response and the follow request will require their approval.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `target_user_id` | string | Yes | ID of the user to follow |
| `user_id` | string | No | Authenticated user's ID. If omitted, resolved automatically via `/users/me` |

**X API:** `POST /2/users/{id}/following` ([docs](https://developer.x.com/en/docs/x-api/users/follows/api-reference/post-users-source_user_id-following))

---

### `x.unfollow_user`

Unfollow a user.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `target_user_id` | string | Yes | ID of the user to unfollow |
| `user_id` | string | No | Authenticated user's ID. If omitted, resolved automatically via `/users/me` |

**X API:** `DELETE /2/users/{id}/following/{target_user_id}` ([docs](https://developer.x.com/en/docs/x-api/users/follows/api-reference/delete-users-source_id-following))

---

### `x.get_followers`

Get a user's followers list (defaults to the authenticated user).

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `user_id` | string | No | authenticated user | User ID to get followers for |
| `max_results` | integer | No | 100 | Maximum results (1-1000) |
| `pagination_token` | string | No | — | Token for fetching the next page (from `meta.next_token` in the response) |

**X API:** `GET /2/users/{id}/followers` ([docs](https://developer.x.com/en/docs/x-api/users/follows/api-reference/get-users-id-followers))

---

### `x.get_following`

Get the list of users that a user follows (defaults to the authenticated user).

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `user_id` | string | No | authenticated user | User ID to get following list for |
| `max_results` | integer | No | 100 | Maximum results (1-1000) |
| `pagination_token` | string | No | — | Token for fetching the next page (from `meta.next_token` in the response) |

**X API:** `GET /2/users/{id}/following` ([docs](https://developer.x.com/en/docs/x-api/users/follows/api-reference/get-users-id-following))

---

### `x.upload_media`

Upload media (image/GIF/video) for use in tweets. Returns a `media_id_string` to pass to `x.post_tweet` via `media_ids`.

**Risk level:** medium

> **Note:** This uses the v1.1 media upload endpoint (`upload.twitter.com/1.1/media/upload.json`) since the X API v2 has not yet published a stable simple-upload replacement. The Bearer token auth format is the same.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `media_data` | string | Yes | Base64-encoded media content (max ~27 MB). |
| `media_category` | string | No | `tweet_image`, `tweet_gif`, or `tweet_video` |
| `alt_text` | string | No | Accessibility alt text (max 1000 characters, images only) |

**Response fields:**
- `media_id_string` — use this value in `x.post_tweet`'s `media_ids` array
- `expires_after_secs` — media is only available for this many seconds; post the tweet promptly

**X API:** `POST https://upload.twitter.com/1.1/media/upload.json` ([docs](https://developer.x.com/en/docs/x-api/v1/media/upload-media/api-reference/post-media-upload))

---

## Auto-Resolving `user_id`

For actions that operate on the authenticated user's own account (`like_tweet`, `unlike_tweet`, `retweet`, `unretweet`, `follow_user`, `unfollow_user`, `get_followers`, `get_following`), the `user_id` parameter is **optional**. If omitted, the connector calls `GET /2/users/me` to look up the authenticated user's ID automatically.

This means agents don't need to know (or be granted) the numeric user ID — they can just call `x.like_tweet` with only `tweet_id` and the connector handles the rest.

## Error Handling

The connector maps X API error responses to typed connector errors:

| HTTP Status | Connector Error | Description |
|-------------|-----------------|-------------|
| 401 | `AuthError` | Invalid or expired token |
| 403 | `AuthError` | Insufficient permissions (check OAuth scopes) |
| 429 | `RateLimitError` | Rate limit exceeded (includes retry-after) |
| Other 4xx/5xx | `ExternalError` | X API error with message |
| Client timeout | `TimeoutError` | Request timed out (30s default) |
| Context cancel | `CanceledError` | Request canceled by caller (not retryable) |

Rate limit responses use the `x-rate-limit-reset` header to determine retry timing (defaults to 30s if missing).

Response body reads are capped at 10 MB to prevent OOM from unexpectedly large responses.

## Adding a New Action

Each action lives in its own file. To add one:

1. Create `connectors/x/<action_name>.go` with an action struct and an `Execute` method.
   - Use shared param structs from `engagement_params.go` where the shape matches.
   - Call `a.conn.resolveUserID(ctx, creds, params.UserID)` if the action operates on a user by ID to support the optional auto-resolve pattern.
2. Use `a.conn.do(ctx, creds, method, path, body, &resp)` for the HTTP lifecycle — it handles Bearer auth, rate limiting, timeouts, and response size limits.
3. Return `connectors.JSONResult(respBody)` to wrap the response.
4. Register the action in `Actions()` inside `x.go`.
5. Add the action to the manifest in `manifest.go`.
6. Add tests in `<action_name>_test.go` using `httptest.NewServer` and `newForTest()`.

## File Structure

```
connectors/x/
├── x.go                    # XConnector struct, New(), Actions(), do(), resolveUserID()
├── manifest.go             # Manifest() — actions, templates, OAuth provider, credentials
├── engagement_params.go    # Shared param structs for engagement actions + validation helpers
├── post_tweet.go           # x.post_tweet action
├── delete_tweet.go         # x.delete_tweet action
├── send_dm.go              # x.send_dm action
├── get_user_tweets.go      # x.get_user_tweets action
├── search_tweets.go        # x.search_tweets action
├── get_me.go               # x.get_me action
├── like_tweet.go           # x.like_tweet and x.unlike_tweet actions
├── retweet.go              # x.retweet and x.unretweet actions
├── follow_user.go          # x.follow_user and x.unfollow_user actions
├── get_followers.go        # x.get_followers and x.get_following actions
├── upload_media.go         # x.upload_media action + doUpload helper
├── register.go             # init() — registers the connector as a built-in
├── x_test.go               # Connector-level tests (ID, Actions, Manifest, ValidateCredentials)
├── helpers_test.go         # Shared test helpers (validCreds, newForTest)
├── *_test.go               # Per-action tests
└── README.md               # This file
```

## Testing

All tests use `httptest.NewServer` to mock the X API — no real API calls are made. Tests run in parallel (`t.Parallel()`) and each uses its own server instance to avoid shared state.

```bash
go test ./connectors/x/... -v
```
