# Asana Connector

The Asana connector integrates Permission Slip with the [Asana REST API](https://developers.asana.com/docs/asana). It uses plain `net/http` with personal access tokens for authentication — no OAuth required.

## Connector ID

`asana`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `api_key` | Yes | An Asana personal access token. Generate one at [Asana Developer Console](https://developers.asana.com/docs/personal-access-token). |
| `workspace_id` | No | Default workspace GID. Used as a fallback by `search_tasks` when `workspace_id` is not provided in the action parameters. |

The credential `auth_type` in the database is `api_key`. Tokens are stored encrypted in Supabase Vault and decrypted only at execution time.

## Actions

### `asana.create_task`

Creates a new task in a project.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `project_id` | string | Yes | Project GID to create the task in |
| `name` | string | Yes | Task name |
| `notes` | string | No | Task description (supports rich text) |
| `assignee` | string | No | Assignee user GID or email |
| `due_on` | string | No | Due date (YYYY-MM-DD) |
| `due_at` | string | No | Due date and time (ISO 8601) |
| `tags` | array | No | Tag GIDs to apply |
| `custom_fields` | object | No | Custom field GID to value mapping |

**Response:**

```json
{
  "gid": "67890",
  "name": "Fix login bug",
  "permalink_url": "https://app.asana.com/0/12345/67890"
}
```

**Asana API:** `POST /tasks` ([docs](https://developers.asana.com/reference/createtask))

---

### `asana.update_task`

Updates fields on an existing task.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `task_id` | string | Yes | Task GID to update |
| `name` | string | No | Updated task name |
| `notes` | string | No | Updated description |
| `assignee` | string | No | Assignee user GID or email |
| `due_on` | string | No | Due date (YYYY-MM-DD) |
| `due_at` | string | No | Due date and time (ISO 8601) |
| `completed` | boolean | No | Whether the task is completed |
| `custom_fields` | object | No | Custom field GID to value mapping |

**Response:**

```json
{
  "gid": "67890",
  "name": "Updated task",
  "permalink_url": "https://app.asana.com/0/1/67890"
}
```

**Asana API:** `PUT /tasks/{task_gid}` ([docs](https://developers.asana.com/reference/updatetask))

---

### `asana.add_comment`

Adds a comment (story) to a task.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `task_id` | string | Yes | Task GID to comment on |
| `text` | string | No* | Plain text comment |
| `html_text` | string | No* | Rich text comment (HTML) |

*At least one of `text` or `html_text` is required.

**Response:**

```json
{
  "gid": "99999",
  "text": "This is a comment"
}
```

**Asana API:** `POST /tasks/{task_gid}/stories` ([docs](https://developers.asana.com/reference/createstoryfortask))

---

### `asana.complete_task`

Marks a task as complete. This is a separate action from `update_task` to provide an explicit permission gate — completing a task is an irreversible-in-practice workflow event.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `task_id` | string | Yes | Task GID to complete |

**Response:**

```json
{
  "gid": "67890",
  "name": "Some task",
  "completed": true,
  "permalink_url": "https://app.asana.com/0/1/67890"
}
```

**Asana API:** `PUT /tasks/{task_gid}` with `{"data": {"completed": true}}` ([docs](https://developers.asana.com/reference/updatetask))

---

### `asana.create_subtask`

Creates a subtask under an existing task. Critical for breaking down work, a core agent use case.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `parent_task_id` | string | Yes | Parent task GID |
| `name` | string | Yes | Subtask name |
| `notes` | string | No | Subtask description |
| `assignee` | string | No | Assignee user GID or email |
| `due_on` | string | No | Due date (YYYY-MM-DD) |

**Response:**

```json
{
  "gid": "11111",
  "name": "Sub-item",
  "permalink_url": "https://app.asana.com/0/1/11111"
}
```

**Asana API:** `POST /tasks/{task_gid}/subtasks` ([docs](https://developers.asana.com/reference/createsubtaskfortask))

---

### `asana.search_tasks`

Searches and filters tasks across projects in a workspace.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `workspace_id` | string | Yes* | Workspace GID to search in |
| `text` | string | No | Full-text search query |
| `assignee` | string | No | Filter by assignee GID or email |
| `projects` | array | No | Filter by project GIDs |
| `completed` | boolean | No | Filter by completion status |
| `due_on_before` | string | No | Filter tasks due before this date (YYYY-MM-DD) |
| `due_on_after` | string | No | Filter tasks due after this date (YYYY-MM-DD) |
| `limit` | integer | No | Maximum results (default 20) |

*Falls back to `workspace_id` from credentials if not provided in parameters.

**Response:**

```json
[
  {
    "gid": "1",
    "name": "Fix login bug",
    "completed": false,
    "permalink_url": "https://app.asana.com/0/1/1"
  }
]
```

**Asana API:** `GET /workspaces/{workspace_gid}/tasks/search` ([docs](https://developers.asana.com/reference/searchtasksforworkspace))

## Error Handling

The connector maps Asana API responses to typed connector errors. Asana error responses include a `help` field which is appended to error messages as a hint.

| Asana Status | Connector Error | HTTP Response |
|--------------|-----------------|---------------|
| 401, 403 | `AuthError` | 502 Bad Gateway |
| 400 | `ValidationError` | 400 Bad Request |
| 404 | `ValidationError` | 400 Bad Request |
| 429 | `RateLimitError` | 429 Too Many Requests |
| Other 4xx/5xx | `ExternalError` | 502 Bad Gateway |
| Client timeout / context deadline | `TimeoutError` | 504 Gateway Timeout |
| Context canceled | `TimeoutError` | 504 Gateway Timeout |

## Architecture

### Request/Response Envelope

Asana wraps all request and response bodies in `{"data": ...}`. The connector handles this transparently:

- **Requests:** `do()` wraps the request body map in `{"data": ...}` before sending
- **Responses:** `do()` unwraps the `{"data": ...}` envelope before unmarshaling into the response struct
- **GET requests:** `doRaw()` skips the request envelope (no body) and returns the raw response

### Shared HTTP Lifecycle

Both `do()` and `doRaw()` delegate to `execRequest()` which handles:
- Bearer token auth from credentials
- HTTP execution with timeout/cancellation detection
- Response body size limiting (10 MB cap)
- Status code → typed error mapping

## File Structure

```
connectors/asana/
├── asana.go                # AsanaConnector struct, New(), Actions(), ValidateCredentials(), do(), doRaw(), execRequest()
├── manifest.go             # Manifest() with 6 action schemas and 6 templates
├── response.go             # checkResponse() — HTTP status → typed error mapping
├── create_task.go          # asana.create_task action
├── update_task.go          # asana.update_task action
├── add_comment.go          # asana.add_comment action
├── complete_task.go        # asana.complete_task action
├── create_subtask.go       # asana.create_subtask action
├── search_tasks.go         # asana.search_tasks action
├── asana_test.go           # Connector-level tests (manifest, envelope, errors)
├── helpers_test.go         # Shared test helpers (validCreds)
├── create_task_test.go     # Tests for asana.create_task
├── update_task_test.go     # Tests for asana.update_task
├── add_comment_test.go     # Tests for asana.add_comment
├── complete_task_test.go   # Tests for asana.complete_task
├── create_subtask_test.go  # Tests for asana.create_subtask
├── search_tasks_test.go    # Tests for asana.search_tasks
└── README.md               # This file
```

## Testing

All tests use `httptest.NewServer` to mock the Asana API — no real API calls are made.

```bash
go test ./connectors/asana/... -v
```
