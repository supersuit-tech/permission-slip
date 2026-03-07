# Figma Connector

The Figma connector integrates Permission Slip with the [Figma REST API](https://www.figma.com/developers/api) for reading design files, exporting images, and managing comments. It uses plain `net/http` with personal access tokens — no third-party SDK.

## Connector ID

`figma`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `personal_access_token` | Yes | A Figma personal access token ([create one here](https://www.figma.com/developers/api#authentication)) |

The credential `auth_type` is `custom`. The token is sent via the `X-Figma-Token` header on every request.

## Actions

### `figma.get_file`

Gets a full design file tree with metadata. Supports filtering by depth and specific node IDs.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `file_key` | string | Yes | — | The file key or full Figma URL (key is auto-extracted from URLs like `https://www.figma.com/design/abc123DEF/...`) |
| `depth` | integer | No | full | How deep to traverse the document tree |
| `node_ids` | string | No | — | Comma-separated list of node IDs to retrieve (e.g. `1:2,3:4`) |

**Figma API:** `GET /v1/files/:file_key` ([docs](https://www.figma.com/developers/api#get-files-endpoint))

---

### `figma.get_components`

Gets design system components from a file.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `file_key` | string | Yes | The file key or full Figma URL |

**Figma API:** `GET /v1/files/:file_key/components`

---

### `figma.export_images`

Exports PNG, SVG, PDF, or JPG from specific nodes in a file. Returns temporary download URLs.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `file_key` | string | Yes | — | The file key or full Figma URL |
| `node_ids` | string | Yes | — | Comma-separated list of node IDs to export (e.g. `1:2,3:4`) |
| `format` | string | Yes | — | Export format: `png`, `svg`, `pdf`, or `jpg` |
| `scale` | number | No | `1` | Image scale factor (0.01–4). Only applies to png/jpg. |

**Figma API:** `GET /v1/images/:file_key` ([docs](https://www.figma.com/developers/api#get-images-endpoint))

---

### `figma.list_comments`

Lists comments on a Figma file.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `file_key` | string | Yes | — | The file key or full Figma URL |
| `as_md` | boolean | No | `false` | If true, return comment messages as markdown |

**Figma API:** `GET /v1/files/:file_key/comments` ([docs](https://www.figma.com/developers/api#get-comments-endpoint))

---

### `figma.post_comment`

Posts a comment on a Figma file, or replies to an existing comment thread.

**Risk level:** medium — creates visible content in the design file

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `file_key` | string | Yes | The file key or full Figma URL |
| `message` | string | Yes | Comment message text |
| `comment_id` | string | No | ID of the comment to reply to (for threaded replies) |

**Figma API:** `POST /v1/files/:file_key/comments` ([docs](https://www.figma.com/developers/api#post-comments-endpoint))

---

### `figma.get_versions`

Gets the version history for a Figma file.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `file_key` | string | Yes | The file key or full Figma URL |

**Figma API:** `GET /v1/files/:file_key/versions` ([docs](https://www.figma.com/developers/api#get-file-versions-endpoint))

## Error Handling

The connector maps Figma API error responses to typed connector errors:

| HTTP Status | Connector Error | Description |
|-------------|----------------|-------------|
| 400 | `ValidationError` | Bad request (invalid parameters) |
| 401 | `AuthError` | Invalid or expired personal access token |
| 403 | `AuthError` | Insufficient permissions |
| 404 | `ValidationError` | File/resource not found or wrong file key |
| 429 | `RateLimitError` | Rate limit exceeded (respects `Retry-After` header, defaults to 30s) |
| 5xx | `ExternalError` | Figma server error |
| Timeout | `TimeoutError` | Request timed out (30s default) |

## Security

- **Credential protection on redirects:** The HTTP client strips the `X-Figma-Token` header from any cross-origin redirect to prevent credential leakage.
- **Response size limit:** Responses are capped at 20 MB to prevent memory exhaustion from large design files.
- **Input validation:** File keys are validated against path traversal sequences. Node IDs are validated against the `X:Y` format.
- **URL extraction:** The `file_key` parameter accepts full Figma URLs (`https://www.figma.com/design/KEY/...`) and auto-extracts the key, so users don't need to manually parse URLs.

## File Structure

```
connectors/figma/
├── figma.go                # Connector struct, HTTP client, validation helpers
├── manifest.go             # ManifestProvider: action schemas, credentials, templates
├── get_file.go             # figma.get_file action
├── get_file_test.go
├── get_components.go       # figma.get_components action
├── get_components_test.go
├── export_images.go        # figma.export_images action
├── export_images_test.go
├── list_comments.go        # figma.list_comments action
├── list_comments_test.go
├── post_comment.go         # figma.post_comment action
├── post_comment_test.go
├── get_versions.go         # figma.get_versions action
├── get_versions_test.go
├── figma_test.go           # Connector-level tests (manifest, HTTP, error mapping)
└── helpers_test.go         # Test helpers (validCreds, etc.)
```

## Configuration Templates

The connector ships with 6 pre-built templates:

| Template | Action | Description |
|----------|--------|-------------|
| Read design file | `figma.get_file` | Agent can read any Figma file's design tree and metadata |
| Get design components | `figma.get_components` | Agent can list components from any Figma file |
| Export images from designs | `figma.export_images` | Agent can export images from any Figma file nodes |
| Read file comments | `figma.list_comments` | Agent can list comments on any Figma file |
| Post comments on designs | `figma.post_comment` | Agent can post comments on any Figma file |
| View version history | `figma.get_versions` | Agent can view version history of any Figma file |
