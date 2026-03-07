# Datadog Connector

The Datadog connector integrates Permission Slip with the [Datadog REST API](https://docs.datadoghq.com/api/latest/). It uses plain `net/http` — no third-party Datadog SDK.

## Connector ID

`datadog`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `api_key` | Yes | A Datadog API key for authenticating requests. |
| `app_key` | Yes | A Datadog Application key for authenticating requests. |

The credential `auth_type` in the database is `custom`. Both keys are required for all Datadog API calls.

## Actions

### `datadog.get_metrics`

Queries time series metrics from Datadog.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `query` | string | Yes | Datadog metrics query (e.g. `avg:system.cpu.user{host:myhost}`) |
| `from` | integer | Yes | Start of query window as UNIX epoch timestamp (seconds) |
| `to` | integer | Yes | End of query window as UNIX epoch timestamp (seconds) |

**Datadog API:** `GET /api/v1/query` ([docs](https://docs.datadoghq.com/api/latest/metrics/#query-timeseries-data-across-multiple-products))

---

### `datadog.create_incident`

Creates a new incident in Datadog.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `title` | string | Yes | — | Incident title |
| `severity` | string | No | `"UNKNOWN"` | One of `SEV-1` through `SEV-5` or `UNKNOWN` |
| `customer_impact_scope` | string | No | — | Description of the customer impact |
| `customer_impacted` | boolean | No | `false` | Whether customers are impacted |

**Datadog API:** `POST /api/v2/incidents` ([docs](https://docs.datadoghq.com/api/latest/incidents/#create-an-incident))

---

### `datadog.snooze_alert`

Mutes (snoozes) a Datadog monitor for a specified duration.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `monitor_id` | integer | Yes | The ID of the monitor to mute |
| `end` | integer | No | UNIX epoch timestamp when the mute should end |
| `scope` | string | No | Scope to apply the mute to (e.g. `host:myhost`) |

**Datadog API:** `POST /api/v1/monitor/{monitor_id}/mute` ([docs](https://docs.datadoghq.com/api/latest/monitors/#mute-a-monitor))

---

### `datadog.trigger_runbook`

Triggers a Datadog Workflow automation (runbook).

**Risk level:** high

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `workflow_id` | string | Yes | The ID of the workflow to trigger |
| `payload` | object | No | Input payload to pass to the workflow |

**Datadog API:** `POST /api/v2/workflows/{workflow_id}/instances` ([docs](https://docs.datadoghq.com/api/latest/workflow-automation/#execute-a-workflow))

## Error Handling

| Datadog Status | Connector Error | HTTP Response |
|----------------|-----------------|---------------|
| 400 | `ValidationError` | 400 Bad Request |
| 401, 403 | `AuthError` | 502 Bad Gateway |
| 429 | `RateLimitError` | 429 Too Many Requests |
| Other 4xx/5xx | `ExternalError` | 502 Bad Gateway |
| Client timeout / context deadline | `TimeoutError` | 504 Gateway Timeout |

## File Structure

```
connectors/datadog/
├── datadog.go              # DatadogConnector struct, New(), Manifest(), do(), ValidateCredentials()
├── get_metrics.go          # datadog.get_metrics action
├── create_incident.go      # datadog.create_incident action
├── snooze_alert.go         # datadog.snooze_alert action
├── trigger_runbook.go      # datadog.trigger_runbook action
├── response.go             # Shared HTTP response → typed error mapping
├── datadog_test.go         # Connector-level tests
├── helpers_test.go         # Shared test helpers (validCreds)
├── get_metrics_test.go     # Get metrics action tests
├── create_incident_test.go # Create incident action tests
├── snooze_alert_test.go    # Snooze alert action tests
├── trigger_runbook_test.go # Trigger runbook action tests
└── README.md               # This file
```

## Testing

All tests use `httptest.NewServer` to mock the Datadog API — no real API calls are made.

```bash
go test ./connectors/datadog/... -v
```
