# Custom Connectors

Custom connectors extend Permission Slip with integrations loaded from external Git repositories. Unlike [built-in connectors](creating-connectors.md) (compiled into the binary), custom connectors are **standalone executables** that communicate via a JSON-over-stdin/stdout protocol.

This means connectors can be written in any language (Go, Python, Node, shell scripts, etc.) — the server spawns them as subprocesses and exchanges JSON messages.

## How It Works

1. You define connector repos in `custom-connectors.json` (gitignored) or in the `CUSTOM_CONNECTORS_JSON` environment variable
2. `make install-connectors` clones each repo into the connectors directory and runs `make build`
3. On server startup, the server scans the connectors directory, reads each `connector.json` manifest, registers them alongside built-in connectors
4. DB rows (`connectors`, `connector_actions`, `connector_required_credentials`) are auto-upserted from the manifest on every startup — no manual migrations or seed steps needed
5. When an action executes, the `ExternalConnector` adapter spawns the binary, writes JSON to stdin, reads the JSON response from stdout

The connectors directory defaults to `~/.permission_slip/connectors/` but can be overridden with the `CONNECTORS_DIR` environment variable (see [Deploying to Heroku / PaaS](#deploying-to-heroku--paas)).

## Quick Start

### 1. Create `custom-connectors.json`

Copy the example file and edit it:

```bash
cp custom-connectors.example.json custom-connectors.json
```

```json
{
  "connectors": [
    {
      "repo": "https://github.com/acme/ps-jira-connector",
      "ref": "main"
    }
  ]
}
```

This file is gitignored — it's user-specific configuration, not checked in.

### 2. Install connectors

```bash
make install-connectors
```

This clones each repo, runs `make build`, and verifies the manifest and executable. Connectors are installed to the connectors directory (`~/.permission_slip/connectors/<repo-name>/` by default, or `CONNECTORS_DIR` if set).

### 3. Start the server

```bash
make dev
```

The server automatically detects installed connectors on startup. If `custom-connectors.json` exists, `make dev` also runs `make install-connectors` first.

## Creating a Custom Connector

### Repository Structure

```
my-connector/
├── connector.json   # Manifest: id, name, actions, required credentials
├── main.go          # Reads JSON from stdin, writes JSON to stdout (any language works)
├── go.mod           # Optional — use whatever language/build system you want
└── Makefile         # "make build" must produce a "connector" executable
```

### connector.json (Manifest)

The manifest describes the connector's identity, actions, and credential requirements:

```json
{
  "id": "jira",
  "name": "Jira",
  "description": "Jira issue tracking integration",
  "actions": [
    {
      "action_type": "jira.create_issue",
      "name": "Create Issue",
      "description": "Create a Jira issue",
      "risk_level": "low",
      "parameters_schema": {
        "type": "object",
        "required": ["base_url", "project_key", "summary"],
        "properties": {
          "base_url": {
            "type": "string",
            "description": "Jira instance URL (e.g. https://mycompany.atlassian.net)"
          },
          "project_key": {
            "type": "string",
            "description": "Project key (e.g. PROJ)"
          },
          "summary": {
            "type": "string",
            "description": "Issue summary"
          },
          "issue_type": {
            "type": "string",
            "enum": ["Task", "Bug", "Story", "Epic"],
            "default": "Task",
            "description": "Issue type"
          }
        }
      }
    }
  ],
  "required_credentials": [
    {
      "service": "jira",
      "auth_type": "api_key",
      "instructions_url": "https://support.atlassian.com/atlassian-account/docs/manage-api-tokens-for-your-atlassian-account/"
    }
  ]
}
```

The `parameters_schema` is a JSON Schema object that describes the action's expected parameters. The frontend uses this to render rich parameter displays in the approval review modal — showing human-readable descriptions, type annotations, enum choices, and default values instead of raw key-value pairs.

**Supported schema properties:**

| Property | Purpose | Example |
|----------|---------|---------|
| `type` | Field data type | `"string"`, `"integer"`, `"boolean"` |
| `description` | Human-readable label shown in the UI | `"Repository owner"` |
| `required` (top-level) | Array of required field names | `["owner", "repo"]` |
| `enum` | Allowed values for the field | `["merge", "squash", "rebase"]` |
| `default` | Default value (shown when value matches) | `"Task"` |

### UI Rendering Hints (`x-ui`)

Connectors can customize how their parameter forms appear in the frontend by adding `x-ui` vendor extensions to `parameters_schema`. This uses the [standard JSON Schema mechanism](https://json-schema.org/understanding-json-schema/reference/generic.html) for custom metadata — no new DB tables, API endpoints, or migrations required.

When `x-ui` is absent, parameters render with the default behavior (text inputs with constraint mode dropdowns). Connectors opt in incrementally — you can add `x-ui` to a single field without touching the rest.

#### Property-level `x-ui` fields

Add `x-ui` to individual properties to control how each field renders:

| Field | Type | Description |
|-------|------|-------------|
| `widget` | `string` | Input widget type (see [Widget types](#widget-types) below). Defaults to `"text"`. |
| `label` | `string` | Display label override. Defaults to the property key. |
| `placeholder` | `string` | Input placeholder text. |
| `group` | `string` | ID of the group this field belongs to (must match a group defined in root-level `x-ui.groups`). |
| `help_url` | `string` | Link to external documentation, rendered as a help icon next to the field. |
| `help_text` | `string` | Additional help text displayed below the field, beyond `description`. |
| `visible_when` | `object` | Conditional visibility: `{ "field": "<key>", "equals": <value> }`. The field is hidden unless the referenced field has the specified value. `<value>` should match the type of the referenced field (string, number, or boolean). |

#### Root-level `x-ui` fields

Add `x-ui` at the schema root to control field ordering and grouping:

| Field | Type | Description |
|-------|------|-------------|
| `groups` | `array` | Field group definitions: `[{ "id", "label", "description?", "collapsed?" }]`. Groups render as collapsible sections. |
| `order` | `string[]` | Explicit field ordering. Fields not listed appear after ordered fields in their original order. |

#### Widget types

| Widget | Renders as | Best for |
|--------|-----------|----------|
| `text` | Standard text input (default) | Free-form strings |
| `select` | Dropdown menu | Fields with `enum` values |
| `textarea` | Multi-line text area | Long-form text, descriptions, message bodies |
| `toggle` | On/off switch | Boolean fields |
| `number` | Numeric input with stepper | Integer or decimal fields |
| `date` | Date picker | Date or datetime fields |

#### Field groups

Groups render as collapsible sections in the parameter form. Each group has:

- **`id`** — Unique identifier referenced by property-level `x-ui.group`
- **`label`** — Display heading for the section
- **`description`** (optional) — Subheading or explanatory text
- **`collapsed`** (optional) — If `true`, the section starts collapsed. Useful for advanced/optional fields.

Fields that don't reference a group appear in an ungrouped section at the top.

#### Conditional visibility (`visible_when`)

Use `visible_when` to show a field only when another field has a specific value:

```json
{
  "type": "object",
  "properties": {
    "send_notification": {
      "type": "boolean",
      "x-ui": { "widget": "toggle", "label": "Send notification" }
    },
    "notification_email": {
      "type": "string",
      "x-ui": {
        "label": "Notification email",
        "visible_when": { "field": "send_notification", "equals": true }
      }
    }
  }
}
```

In this example, the "Notification email" field only appears when "Send notification" is toggled on.

**Note:** Fields with `visible_when` should **not** be listed in the schema's `required` array. When a field is hidden, its value won't be submitted — but JSON Schema validation would still flag it as missing. Use application-level validation instead if the field is conditionally required.

#### Full example

```json
{
  "type": "object",
  "required": ["customer_id"],
  "x-ui": {
    "groups": [
      { "id": "billing", "label": "Billing" },
      { "id": "options", "label": "Options", "collapsed": true }
    ],
    "order": ["customer_id", "currency", "auto_advance"]
  },
  "properties": {
    "customer_id": {
      "type": "string",
      "x-ui": { "label": "Customer", "placeholder": "cus_ABC123", "group": "billing" }
    },
    "currency": {
      "type": "string",
      "enum": ["usd", "eur", "gbp"],
      "x-ui": { "widget": "select", "group": "billing" }
    },
    "auto_advance": {
      "type": "boolean",
      "x-ui": { "widget": "toggle", "label": "Auto-send invoice", "group": "options" }
    }
  }
}
```

This renders:
1. A **Billing** section (expanded) with "Customer" (text input with placeholder) and "currency" (dropdown)
2. An **Options** section (collapsed by default) with an "Auto-send invoice" toggle

**Manifest rules:**

| Field | Requirements |
|-------|-------------|
| `id` | Lowercase alphanumeric with hyphens/underscores, 1-63 chars |
| `name` | Human-readable display name (required) |
| `actions[].action_type` | Must be prefixed with the connector `id` + `.` (e.g., `jira.create_issue`) |
| `actions[].risk_level` | `low`, `medium`, or `high` (optional) |
| `actions[].parameters_schema` | JSON Schema describing expected parameters (optional, recommended) |
| `required_credentials[].auth_type` | `api_key`, `basic`, or `custom` |
| `required_credentials[].instructions_url` | URL to human-readable credential setup docs (optional, must be http/https, max 2048 chars) |

### Makefile

Your `Makefile` must have a `build` target that produces an executable named `connector` in the repo root:

```makefile
build:
	go build -o connector .
```

For Python connectors, the `connector` file can be a wrapper script:

```makefile
build:
	chmod +x connector
```

Where `connector` is:

```bash
#!/usr/bin/env python3
import my_connector
my_connector.main()
```

### Execution Protocol (JSON over stdin/stdout)

The server sends a JSON object to stdin and reads a JSON response from stdout. Stderr is captured for error reporting but not parsed.

#### Execute an action

```
→ stdin:  {"command":"execute","action_type":"jira.create_issue","parameters":{...},"credentials":{"api_key":"..."}}
← stdout: {"success":true,"data":{"issue_key":"PROJ-123"}}
```

#### Validate credentials

```
→ stdin:  {"command":"validate_credentials","credentials":{"api_key":"..."}}
← stdout: {"success":true}
```

#### Error response

```
← stdout: {"success":false,"error":{"type":"validation","message":"missing required field: project_key"}}
```

**Error types:**

| Type | Connector Error | HTTP Status |
|------|----------------|-------------|
| `validation` | `ValidationError` | 400 |
| `auth` | `AuthError` | 502 |
| `external` | `ExternalError` | 502 |
| `rate_limit` | `RateLimitError` | 429 |
| `timeout` | `TimeoutError` | 504 |

### Example: Minimal Go Connector

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type request struct {
	Command     string            `json:"command"`
	ActionType  string            `json:"action_type"`
	Parameters  json.RawMessage   `json:"parameters"`
	Credentials map[string]string `json:"credentials"`
}

type response struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   *errorDetail    `json:"error,omitempty"`
}

type errorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func main() {
	var req request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		fmt.Fprintf(os.Stderr, "failed to decode request: %v\n", err)
		os.Exit(1)
	}

	switch req.Command {
	case "execute":
		data, _ := json.Marshal(map[string]string{"status": "done"})
		json.NewEncoder(os.Stdout).Encode(response{Success: true, Data: data})

	case "validate_credentials":
		if req.Credentials["api_key"] == "" {
			json.NewEncoder(os.Stdout).Encode(response{
				Success: false,
				Error:   &errorDetail{Type: "validation", Message: "api_key is required"},
			})
			return
		}
		json.NewEncoder(os.Stdout).Encode(response{Success: true})

	default:
		json.NewEncoder(os.Stdout).Encode(response{
			Success: false,
			Error:   &errorDetail{Type: "validation", Message: "unknown command: " + req.Command},
		})
	}
}
```

## Community Connectors

Looking for connectors built by the community? See the [Community Connectors](community-connectors.md) directory.

## Security Considerations

External connectors run as subprocesses with the same privileges as the server process. Treat custom connectors with the same trust level as any code dependency:

- Only install connectors from repos you trust
- Review the connector source code before installation
- Credentials are passed to the subprocess via stdin (not environment variables or command-line arguments) to reduce exposure surface

## Deploying to Heroku / PaaS

On platforms with ephemeral filesystems (Heroku, Railway, Render, etc.), you can't rely on a writable home directory for persistence. Instead, install connectors during the **build phase** so they become part of the deployed artifact.

### Environment variables

| Variable | Purpose |
|----------|---------|
| `CONNECTORS_DIR` | Directory where connectors are installed and loaded from. Defaults to `~/.permission_slip/connectors/`. Set to a path inside your app (e.g. `/app/connectors`) on PaaS platforms. |
| `CUSTOM_CONNECTORS_JSON` | Inline JSON connector config. When set, `install-connectors` reads from this instead of `custom-connectors.json` on disk. |

### Heroku example

```bash
# Define your connectors as a config var
heroku config:set CUSTOM_CONNECTORS_JSON='{"connectors":[{"repo":"https://github.com/acme/ps-jira-connector","ref":"v1.2.0"}]}'

# Point the server at a path inside the slug
heroku config:set CONNECTORS_DIR=/app/connectors
```

Then add a build step that runs the installer. For example, create `bin/post_compile`:

```bash
#!/usr/bin/env bash
set -euo pipefail
echo "-----> Installing custom connectors"
make install-connectors
```

The installer reads `CUSTOM_CONNECTORS_JSON`, clones and builds each connector into `CONNECTORS_DIR`, and the resulting binaries become part of the Heroku slug. On startup, the server loads them from the same path.

### Docker example

```dockerfile
# Build stage — install connectors
ENV CONNECTORS_DIR=/app/connectors
ENV CUSTOM_CONNECTORS_JSON='{"connectors":[{"repo":"https://github.com/acme/ps-jira-connector","ref":"v1.2.0"}]}'
RUN make install-connectors

# Runtime — server reads from the same path
ENV CONNECTORS_DIR=/app/connectors
```

## Troubleshooting

**"connector executable not found"** — Run `make install-connectors` to build the connector binary.

**"connector returned invalid JSON"** — Ensure your connector writes valid JSON to stdout and does not print debug output to stdout (use stderr for logging).

**"connector process failed"** — Check stderr output in the server logs. The connector may have crashed or returned a non-zero exit code.

**Connector not appearing after install** — Verify `connector.json` exists and is valid in the connector's subdirectory (check `CONNECTORS_DIR` or the default `~/.permission_slip/connectors/<name>/`). The server logs connector loading at startup.
