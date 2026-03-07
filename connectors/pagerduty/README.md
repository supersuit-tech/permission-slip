# PagerDuty Connector

The PagerDuty connector integrates Permission Slip with the [PagerDuty REST API](https://developer.pagerduty.com/api-reference/). It uses plain `net/http` — no third-party PagerDuty SDK.

## Connector ID

`pagerduty`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `api_key` | Yes | A PagerDuty REST API key (v2). |

The credential `auth_type` in the database is `api_key`. Tokens are stored encrypted and decrypted only at execution time.

## Actions

### `pagerduty.create_incident`

Creates a new incident in PagerDuty.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `service_id` | string | Yes | The ID of the PagerDuty service |
| `title` | string | Yes | Incident title |
| `body` | string | No | Incident body/details |
| `urgency` | string | No | `high` or `low` |
| `escalation_policy_id` | string | No | Override the escalation policy |

**PagerDuty API:** `POST /incidents` ([docs](https://developer.pagerduty.com/api-reference/b3A6Mjc0ODEzMg-create-an-incident))

---

### `pagerduty.acknowledge_alert`

Acknowledges an incident to indicate it is being worked on.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `incident_id` | string | Yes | The ID of the incident to acknowledge |

**PagerDuty API:** `PUT /incidents/{id}` ([docs](https://developer.pagerduty.com/api-reference/b3A6Mjc0ODEzNA-update-an-incident))

---

### `pagerduty.resolve_incident`

Resolves an incident in PagerDuty.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `incident_id` | string | Yes | The ID of the incident to resolve |

**PagerDuty API:** `PUT /incidents/{id}` ([docs](https://developer.pagerduty.com/api-reference/b3A6Mjc0ODEzNA-update-an-incident))

---

### `pagerduty.escalate_incident`

Escalates an incident to the next level in the escalation policy.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `incident_id` | string | Yes | The ID of the incident to escalate |
| `escalation_level` | integer | Yes | The escalation level to set |
| `escalation_policy_id` | string | No | Override the escalation policy |

**PagerDuty API:** `PUT /incidents/{id}` ([docs](https://developer.pagerduty.com/api-reference/b3A6Mjc0ODEzNA-update-an-incident))

---

### `pagerduty.list_on_call`

Lists current on-call entries, optionally filtered by schedule or escalation policy.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `schedule_ids` | array of strings | No | Filter by schedule IDs |
| `escalation_policy_ids` | array of strings | No | Filter by escalation policy IDs |
| `since` | string | No | Start of time range (ISO 8601) |
| `until` | string | No | End of time range (ISO 8601) |

**PagerDuty API:** `GET /oncalls` ([docs](https://developer.pagerduty.com/api-reference/b3A6Mjc0ODE1Ng-list-all-of-the-on-calls))

---

### `pagerduty.add_note`

Adds a note to an existing incident's timeline.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `incident_id` | string | Yes | The ID of the incident |
| `content` | string | Yes | The note content |

**PagerDuty API:** `POST /incidents/{id}/notes` ([docs](https://developer.pagerduty.com/api-reference/b3A6Mjc0ODEzNw-create-a-note-on-an-incident))

## Error Handling

| PagerDuty Status | Connector Error | HTTP Response |
|------------------|-----------------|---------------|
| 400 | `ValidationError` | 400 Bad Request |
| 401, 403 | `AuthError` | 502 Bad Gateway |
| 429 | `RateLimitError` | 429 Too Many Requests |
| Other 4xx/5xx | `ExternalError` | 502 Bad Gateway |
| Client timeout / context deadline | `TimeoutError` | 504 Gateway Timeout |

## File Structure

```
connectors/pagerduty/
├── pagerduty.go              # PagerDutyConnector struct, New(), Manifest(), do(), ValidateCredentials()
├── create_incident.go        # pagerduty.create_incident action
├── acknowledge_alert.go      # pagerduty.acknowledge_alert action
├── resolve_incident.go       # pagerduty.resolve_incident action
├── escalate_incident.go      # pagerduty.escalate_incident action
├── list_on_call.go           # pagerduty.list_on_call action
├── add_note.go               # pagerduty.add_note action
├── response.go               # Shared HTTP response → typed error mapping
├── pagerduty_test.go         # Connector-level tests
├── helpers_test.go           # Shared test helpers (validCreds)
├── create_incident_test.go   # Create incident action tests
├── acknowledge_alert_test.go # Acknowledge alert action tests
├── resolve_incident_test.go  # Resolve incident action tests
├── escalate_incident_test.go # Escalate incident action tests
├── list_on_call_test.go      # List on-call action tests
├── add_note_test.go          # Add note action tests
└── README.md                 # This file
```

## Testing

All tests use `httptest.NewServer` to mock the PagerDuty API — no real API calls are made.

```bash
go test ./connectors/pagerduty/... -v
```
