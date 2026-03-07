# Meta Connector

The Meta connector integrates Permission Slip with [Facebook Pages](https://developers.facebook.com/docs/pages-api) and [Instagram](https://developers.facebook.com/docs/instagram-api) (Business/Creator accounts) using the Meta Graph API v19.0. It uses plain `net/http` with OAuth 2.0 access tokens provided by the platform — no third-party Meta SDK.

## Connector ID

`meta`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `access_token` | Yes | An OAuth 2.0 access token provided automatically by the platform's OAuth infrastructure. |

The credential `auth_type` is `oauth2` with `oauth_provider` set to `meta` (a built-in provider). The platform handles the full OAuth flow, token storage, and automatic refresh — the connector just receives a valid access token at execution time.

**OAuth scopes requested:**

| Scope | Used by |
|-------|---------|
| `pages_manage_posts` | `meta.create_page_post`, `meta.delete_page_post`, `meta.reply_page_comment` |
| `pages_read_engagement` | `meta.list_page_posts` |
| `pages_read_user_content` | `meta.list_page_posts`, `meta.reply_page_comment` |
| `instagram_basic` | `meta.get_instagram_insights` |
| `instagram_content_publish` | `meta.create_instagram_post` |
| `instagram_manage_insights` | `meta.get_instagram_insights` |

## Actions

### `meta.create_page_post`

Creates a post on a Facebook Page.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `page_id` | string | Yes | — | Facebook Page ID |
| `message` | string | Yes | — | Post text content |
| `link` | string | No | — | URL to share with the post |
| `published` | boolean | No | `true` | Whether to publish immediately |

**Response:**

```json
{
  "id": "123456_789012"
}
```

**Graph API:** `POST /{page_id}/feed` ([docs](https://developers.facebook.com/docs/pages-api/posts#create))

**Security notes:**
- The `link` parameter is validated as a proper HTTP/HTTPS URL to prevent injection of malicious URIs.

---

### `meta.delete_page_post`

Deletes a Facebook Page post. This action is **irreversible**.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `post_id` | string | Yes | Post ID to delete (format: `pageId_postId`) |

**Response:**

```json
{
  "status": "deleted",
  "post_id": "123456_789012"
}
```

**Graph API:** `DELETE /{post_id}` ([docs](https://developers.facebook.com/docs/pages-api/posts#delete))

---

### `meta.reply_page_comment`

Replies to a comment on a Facebook Page post.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `comment_id` | string | Yes | Comment ID to reply to |
| `message` | string | Yes | Reply text content |

**Response:**

```json
{
  "id": "reply_456"
}
```

**Graph API:** `POST /{comment_id}/comments` ([docs](https://developers.facebook.com/docs/graph-api/reference/comment))

---

### `meta.create_instagram_post`

Publishes a photo post to Instagram. The image must be hosted at a publicly accessible URL.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `instagram_account_id` | string | Yes | Instagram Business/Creator account ID |
| `image_url` | string | Yes | Public HTTPS URL of the image to post |
| `caption` | string | Yes | Post caption (max 2,200 characters including hashtags) |
| `hashtags` | string | No | Hashtags to append to caption (e.g., `#travel #photo`) |

**Response:**

```json
{
  "id": "media_789",
  "container_id": "container_456"
}
```

**Graph API:** Two-step content publishing process:
1. `POST /{ig_account_id}/media` — Create a media container ([docs](https://developers.facebook.com/docs/instagram-platform/instagram-api-with-instagram-login/content-publishing#step-1--create-a-container))
2. `POST /{ig_account_id}/media_publish` — Publish the container ([docs](https://developers.facebook.com/docs/instagram-platform/instagram-api-with-instagram-login/content-publishing#step-3--publish-the-container))

**Implementation details:**
- Instagram content publishing is asynchronous. After creating a container, the connector polls `GET /{container_id}?fields=status_code` until the status is `FINISHED` (up to 15 polls at 2-second intervals).
- If the container status is `ERROR` or `EXPIRED`, the action fails with an `ExternalError`.
- The `image_url` must be HTTPS and publicly accessible — Instagram's servers fetch the image directly.

**Security notes:**
- `image_url` is validated as an HTTPS URL. Non-HTTPS URLs are rejected because Instagram requires HTTPS for media container creation.
- Caption length is validated client-side (2,200 chars including hashtags) to provide a clear error before hitting the API.

---

### `meta.get_instagram_insights`

Retrieves account-level insights for an Instagram Business/Creator account.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `instagram_account_id` | string | Yes | — | Instagram Business/Creator account ID |
| `metric` | string | No | `impressions` | Metric to retrieve: `impressions`, `reach`, or `profile_views` |
| `period` | string | No | `day` | Time period: `day`, `week`, or `days_28` |

**Response:**

```json
{
  "data": [
    {
      "name": "reach",
      "period": "week",
      "title": "Reach",
      "values": [
        {
          "value": 1500,
          "end_time": "2026-03-07T08:00:00+0000"
        }
      ]
    }
  ]
}
```

**Graph API:** `GET /{ig_account_id}/insights` ([docs](https://developers.facebook.com/docs/instagram-platform/instagram-api-with-instagram-login/insights))

**Validation:**
- `metric` and `period` are validated against allowlists to prevent invalid API requests.

---

### `meta.list_page_posts`

Lists recent posts on a Facebook Page with engagement metrics (likes, comments, shares).

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `page_id` | string | Yes | — | Facebook Page ID |
| `limit` | integer | No | `10` | Maximum number of posts to return (1-100) |
| `since` | integer | No | — | Unix timestamp — only return posts after this time |
| `until` | integer | No | — | Unix timestamp — only return posts before this time |

**Response:**

```json
{
  "data": [
    {
      "id": "page_123_post_1",
      "message": "Hello world",
      "created_time": "2026-03-07T12:00:00+0000",
      "shares": {"count": 5},
      "likes": {"summary": {"total_count": 42}},
      "comments": {"summary": {"total_count": 3}}
    }
  ]
}
```

**Graph API:** `GET /{page_id}/posts` ([docs](https://developers.facebook.com/docs/pages-api/posts#read))

Fields requested: `id,message,created_time,shares,likes.summary(true),comments.summary(true)` — this provides engagement metrics without requiring additional API calls.

---

## Error Handling

The connector maps Meta Graph API error codes and HTTP status codes to typed connector errors:

| Meta Error Code | HTTP Status | Connector Error | Meaning |
|----------------|-------------|-----------------|---------|
| 190 | 401 | `AuthError` | Invalid or expired access token |
| — | 403 | `AuthError` | Missing required permission/scope |
| 4 | 429 | `RateLimitError` | API rate limit exceeded (60s retry) |
| 100 | 400 | `ValidationError` | Invalid parameter sent to the API |
| Other | 4xx/5xx | `ExternalError` | General API error |
| — | timeout | `TimeoutError` | Request didn't complete within 30s |

Response bodies are capped at 10 MB via `io.LimitReader` to prevent out-of-memory from unexpectedly large responses.

## Templates

| Template | Action | What's locked |
|----------|--------|---------------|
| Post to any Facebook Page | `create_page_post` | Nothing — agent controls all parameters |
| Post to specific Facebook Page | `create_page_post` | `page_id` locked; agent controls message |
| Delete Facebook Page posts | `delete_page_post` | Nothing — agent controls post ID |
| Reply to page comments | `reply_page_comment` | Nothing — agent controls comment ID and message |
| Post to Instagram | `create_instagram_post` | Nothing — agent controls all parameters |
| Post to specific Instagram account | `create_instagram_post` | `instagram_account_id` locked; agent controls image and caption |
| View Instagram insights | `get_instagram_insights` | Nothing — agent controls account, metric, and period |
| List Facebook Page posts | `list_page_posts` | Nothing — agent controls page ID and filters |
| List posts from specific page | `list_page_posts` | `page_id` locked; agent controls time range |

## Adding a New Action

Each action lives in its own file. To add one (e.g., `meta.create_instagram_story`):

1. Create `connectors/meta/create_instagram_story.go` with a params struct, `validate()`, and an `Execute` method.
2. Use `a.conn.doJSON(ctx, creds, method, url, reqBody, &resp)` for JSON API calls — it handles marshaling, Bearer auth, rate limiting, response size limits, and timeout detection. For GET requests, use `a.conn.doGet()`. For DELETE, use `a.conn.doDelete()`.
3. Return `connectors.JSONResult(respBody)` to wrap the response into an `ActionResult`.
4. Register the action in `Actions()` inside `meta.go`.
5. Add the action to `Manifest()` inside `manifest.go` with a `ParametersSchema`.
6. Add tests in `create_instagram_story_test.go` using `httptest.NewServer` and `newForTest()`.

## File Structure

```
connectors/meta/
├── meta.go                           # MetaConnector struct, New(), Actions(), doJSON(), doGet(), doDelete(), ValidateCredentials()
├── manifest.go                       # Manifest() — actions, credentials, templates
├── create_page_post.go               # meta.create_page_post action
├── delete_page_post.go               # meta.delete_page_post action
├── reply_page_comment.go             # meta.reply_page_comment action
├── create_instagram_post.go          # meta.create_instagram_post action (async container publishing)
├── get_instagram_insights.go         # meta.get_instagram_insights action
├── list_page_posts.go                # meta.list_page_posts action
├── meta_test.go                      # Connector-level tests (ID, Actions, Manifest, ValidateCredentials, error mapping)
├── helpers_test.go                   # Shared test helpers (validCreds)
├── create_page_post_test.go          # Create page post tests (success, validation, auth, rate limit)
├── delete_page_post_test.go          # Delete page post tests
├── reply_page_comment_test.go        # Reply to comment tests
├── create_instagram_post_test.go     # Instagram post tests (container lifecycle, caption length)
├── get_instagram_insights_test.go    # Instagram insights tests (metric/period validation)
├── list_page_posts_test.go           # List page posts tests (time range, default limit)
└── README.md                         # This file
```

## Testing

All tests use `httptest.NewServer` to mock the Meta Graph API — no real API calls are made.

```bash
go test ./connectors/meta/... -v
go test ./connectors/meta/... -race  # verify no race conditions
```
