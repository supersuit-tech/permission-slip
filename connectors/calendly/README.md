# Calendly Connector

Calendly integration for scheduling, event types, and availability using the [Calendly API v2](https://developer.calendly.com/api-docs).

## Authentication

**Auth type:** `api_key` (personal access token)

Generate a personal access token at:
https://developer.calendly.com/how-to-authenticate-with-personal-access-tokens

The token is sent as a `Bearer` token in the `Authorization` header.

## Actions

| Action | Risk | Description |
|--------|------|-------------|
| `calendly.list_event_types` | low | List scheduling event types (e.g., "30 min meeting") |
| `calendly.create_scheduling_link` | low | Generate a single-use or reusable scheduling link |
| `calendly.list_scheduled_events` | low | List upcoming/past events with filters and guest info |
| `calendly.cancel_event` | medium | Cancel a scheduled event (sends cancellation email) |
| `calendly.get_event` | low | Get full event details including attendees and location |
| `calendly.list_available_times` | low | Check available time slots for a date range |

## User URI

Most Calendly list endpoints require a `user_uri` parameter (a full URI like `https://api.calendly.com/users/abc123`). The connector automatically fetches this from `GET /users/me` when needed. To avoid the extra API call, you can pass `user_uri` directly to `list_event_types` and `list_scheduled_events`.

## Rate Limits

Calendly allows 140 requests per 60 seconds. The connector maps 429 responses to `RateLimitError` with the `Retry-After` header value.
