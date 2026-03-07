# Zoom Connector

The Zoom connector integrates Permission Slip with the [Zoom REST API v2](https://developers.zoom.us/docs/api/) for managing meetings, recordings, and participants. It uses plain `net/http` with OAuth 2.0 access tokens provided by the platform — no third-party Zoom SDK.

## Connector ID

`zoom`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `access_token` | Yes | An OAuth 2.0 access token provided automatically by the platform's OAuth infrastructure. |

The credential `auth_type` is `oauth2` with `oauth_provider` set to `zoom` (a built-in provider). The platform handles the full OAuth flow, token storage, and automatic refresh — the connector just receives a valid access token at execution time.

**OAuth scopes requested:**

| Scope | Used by |
|-------|---------|
| `meeting:read` | `zoom.list_meetings`, `zoom.get_meeting`, `zoom.get_meeting_participants` |
| `meeting:write` | `zoom.create_meeting`, `zoom.update_meeting`, `zoom.delete_meeting` |
| `recording:read` | `zoom.list_recordings` |
| `user:read` | General user context for API calls |

## Actions

### `zoom.list_meetings`

Lists meetings for the authenticated user filtered by type.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `type` | string | No | `upcoming` | Meeting type filter: `scheduled`, `live`, or `upcoming` |
| `page_size` | integer | No | `30` | Number of meetings to return (1-300) |

**Response:**

```json
{
  "total_meetings": 2,
  "meetings": [
    {
      "id": 123456789,
      "uuid": "abc123==",
      "topic": "Team Standup",
      "type": 2,
      "start_time": "2024-01-15T09:00:00Z",
      "duration": 30,
      "timezone": "America/New_York",
      "join_url": "https://zoom.us/j/123456789",
      "status": "waiting"
    }
  ]
}
```

**Zoom API:** `GET /users/me/meetings` ([docs](https://developers.zoom.us/docs/api/rest/reference/zoom-api/methods/#operation/meetings))

---

### `zoom.create_meeting`

Schedules a new Zoom meeting and returns the join URL.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `topic` | string | Yes | — | Meeting topic/title |
| `type` | integer | No | `2` | Meeting type: `1` (instant) or `2` (scheduled) |
| `start_time` | string | No | — | Start time in ISO 8601 format |
| `duration` | integer | No | — | Duration in minutes |
| `timezone` | string | No | — | IANA timezone (e.g. `America/New_York`) |
| `agenda` | string | No | — | Meeting agenda/description |
| `settings` | object | No | — | Meeting settings (see below) |

**Settings object:**

| Name | Type | Description |
|------|------|-------------|
| `join_before_host` | boolean | Allow participants to join before host |
| `waiting_room` | boolean | Enable waiting room |

**Response:**

```json
{
  "id": 123456789,
  "uuid": "abc123==",
  "topic": "Sprint Planning",
  "type": 2,
  "start_time": "2024-01-20T14:00:00Z",
  "duration": 60,
  "timezone": "America/New_York",
  "join_url": "https://zoom.us/j/123456789",
  "password": "abc123"
}
```

**Zoom API:** `POST /users/me/meetings` ([docs](https://developers.zoom.us/docs/api/rest/reference/zoom-api/methods/#operation/meetingCreate))

**Security notes:**
- The `start_url` is intentionally excluded from the response. It contains an embedded token that grants host-level privileges (start meeting, manage participants). Only the participant `join_url` is returned.

---

### `zoom.get_meeting`

Gets full details of a specific meeting.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `meeting_id` | string | Yes | The meeting ID or UUID to retrieve |

**Response:**

```json
{
  "id": 123456789,
  "uuid": "abc123==",
  "topic": "Sprint Planning",
  "type": 2,
  "start_time": "2024-01-20T14:00:00Z",
  "duration": 60,
  "timezone": "America/New_York",
  "agenda": "Review Q1 goals",
  "join_url": "https://zoom.us/j/123456789",
  "password": "abc123",
  "status": "waiting",
  "settings": {
    "join_before_host": true,
    "waiting_room": false,
    "mute_upon_entry": false
  }
}
```

**Zoom API:** `GET /meetings/{meetingId}` ([docs](https://developers.zoom.us/docs/api/rest/reference/zoom-api/methods/#operation/meeting))

**Notes:**
- Meeting IDs containing special characters (e.g., double-encoded UUIDs with `/`) are automatically URL-escaped.

---

### `zoom.update_meeting`

Updates an existing scheduled meeting.

**Risk level:** medium — may notify participants of changes

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `meeting_id` | string | Yes | The meeting ID to update |
| `topic` | string | No | Updated meeting topic |
| `start_time` | string | No | Updated start time (ISO 8601) |
| `duration` | integer | No | Updated duration in minutes |
| `timezone` | string | No | Updated timezone |
| `agenda` | string | No | Updated agenda |
| `settings` | object | No | Updated settings |

**Response:**

```json
{
  "meeting_id": "123456789",
  "updated": true
}
```

**Zoom API:** `PATCH /meetings/{meetingId}` (returns 204 No Content)

---

### `zoom.delete_meeting`

Deletes/cancels a scheduled meeting.

**Risk level:** medium — cancels for all participants

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `meeting_id` | string | Yes | The meeting ID to delete |
| `schedule_for_reminder` | boolean | No | Send cancellation reminder to participants |

**Response:**

```json
{
  "meeting_id": "123456789",
  "deleted": true
}
```

**Zoom API:** `DELETE /meetings/{meetingId}` (returns 204 No Content)

---

### `zoom.list_recordings`

Lists cloud recordings for the authenticated user within a date range.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `from` | string | Yes | Start date in `YYYY-MM-DD` format |
| `to` | string | Yes | End date in `YYYY-MM-DD` format |
| `page_size` | integer | No | Number of recordings per page (1-300, default 30) |

**Response:**

```json
{
  "total_recordings": 1,
  "recordings": [
    {
      "meeting_id": 123456789,
      "uuid": "abc123==",
      "topic": "Team Standup",
      "start_time": "2024-01-15T09:00:00Z",
      "duration": 30,
      "total_size": 1024000,
      "recording_files": [
        {
          "id": "file-1",
          "file_type": "MP4",
          "file_size": 512000,
          "download_url": "https://zoom.us/rec/download/...",
          "play_url": "https://zoom.us/rec/play/...",
          "recording_type": "shared_screen_with_speaker_view",
          "status": "completed"
        }
      ]
    }
  ]
}
```

**Zoom API:** `GET /users/me/recordings` ([docs](https://developers.zoom.us/docs/api/rest/reference/zoom-api/methods/#operation/recordingsList))

---

### `zoom.get_meeting_participants`

Gets the participant list for a past meeting.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `meeting_id` | string | Yes | The meeting ID to get participants for |

**Response:**

```json
{
  "total_participants": 3,
  "participants": [
    {
      "id": "p-1",
      "name": "Alice Smith",
      "email": "alice@example.com",
      "join_time": "2024-01-15T09:00:00Z",
      "leave_time": "2024-01-15T09:30:00Z",
      "duration": 1800
    }
  ]
}
```

**Zoom API:** `GET /past_meetings/{meetingId}/participants` ([docs](https://developers.zoom.us/docs/api/rest/reference/zoom-api/methods/#operation/pastMeetingParticipants))

**Notes:**
- Only works for meetings that have already ended. Returns a 404 error for ongoing or future meetings.

## Error Handling

The connector maps Zoom API error responses to typed connector errors:

| HTTP Status | Connector Error | Description |
|-------------|----------------|-------------|
| 400 | `ValidationError` | Bad request (invalid parameters) |
| 401 | `AuthError` | Invalid or expired access token |
| 403 | `AuthError` | Insufficient permissions/scopes |
| 404 | `ValidationError` | Meeting/resource not found |
| 409 | `ValidationError` | Conflict (e.g., operating on an ended meeting) |
| 429 | `RateLimitError` | Rate limit exceeded (respects `Retry-After` header) |
| 5xx | `ExternalError` | Zoom server error |
| Timeout | `TimeoutError` | Request timed out (30s default) |

## File Structure

```
connectors/zoom/
├── zoom.go                          # Connector struct, HTTP client, error handling
├── manifest.go                      # ManifestProvider: action schemas, credentials, templates
├── list_meetings.go                 # zoom.list_meetings action
├── list_meetings_test.go
├── create_meeting.go                # zoom.create_meeting action
├── create_meeting_test.go
├── get_meeting.go                   # zoom.get_meeting action
├── get_meeting_test.go
├── update_meeting.go                # zoom.update_meeting action
├── update_meeting_test.go
├── delete_meeting.go                # zoom.delete_meeting action
├── delete_meeting_test.go
├── list_recordings.go               # zoom.list_recordings action
├── list_recordings_test.go
├── get_meeting_participants.go      # zoom.get_meeting_participants action
├── get_meeting_participants_test.go
└── helpers_test.go                  # Test helpers (validCreds, etc.)
```

## Configuration Templates

The connector ships with 8 pre-built templates:

| Template | Action | Description |
|----------|--------|-------------|
| List upcoming meetings | `zoom.list_meetings` | Agent can list upcoming meetings |
| Schedule a Zoom meeting | `zoom.create_meeting` | Agent can schedule meetings with any settings |
| Schedule a 30-min Zoom call | `zoom.create_meeting` | Agent can schedule 30-min meetings only |
| View meeting details | `zoom.get_meeting` | Agent can view any meeting's details |
| Update meetings | `zoom.update_meeting` | Agent can update scheduled meetings |
| Cancel meetings | `zoom.delete_meeting` | Agent can cancel scheduled meetings |
| Find recordings | `zoom.list_recordings` | Agent can search recordings by date range |
| View participants | `zoom.get_meeting_participants` | Agent can view past meeting participants |
