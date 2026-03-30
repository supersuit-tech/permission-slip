# Permission Slip — OpenAPI 3.0 Specification

This directory contains the complete OpenAPI 3.0 specification for the Permission Slip API.

## Overview

Permission Slip is the approval layer for [Openclaw](https://openclaw.org). Openclaw submits pre-defined **actions** for human approval, and Permission Slip executes them using the user's stored credentials. Actions are the core primitive — structured templates (like `email.send`, `flight.book`, `payment.charge`) that define what Openclaw can request, with validated parameters and human-readable display formats.

This specification defines the HTTP API that Openclaw uses to interact with Permission Slip at `http://localhost:8080` (the local dev default). Openclaw integrates with this single endpoint for everything — action discovery, registration, approval, and execution.

## Structure

The OpenAPI specification is modularized for maintainability and reusability:

```
spec/openapi/
├── openapi.yaml                 # Main OpenAPI document (entry point)
├── openapi.bundle.yaml          # Single-file bundled version (auto-generated)
└── components/
    ├── schemas/                 # Data models
    │   ├── registration.yaml    # Agent registration schemas
    │   ├── approvals.yaml       # Approval request/response schemas
    │   ├── actions.yaml         # Action execution schemas
    │   ├── action_config_templates.yaml # Action config template schemas
    │   ├── action_configurations.yaml   # Action configuration schemas
    │   ├── agents.yaml          # Agent management schemas
    │   ├── agent_connectors.yaml # Agent-connector relationship schemas
    │   ├── audit_events.yaml    # Audit event and log schemas
    │   ├── billing.yaml         # Billing, plan, subscription, usage, invoice schemas
    │   ├── capabilities.yaml    # Agent capability discovery schemas
    │   ├── config.yaml          # Server configuration schemas
    │   ├── connectors.yaml      # Connector/integration schemas
    │   ├── credentials.yaml     # Credential vault schemas
    │   ├── onboarding.yaml      # Onboarding workflow schemas
    │   ├── profiles.yaml        # User profile schemas
    │   ├── push_subscriptions.yaml # Web Push subscription schemas
    │   ├── registration_invites.yaml # Registration invite schemas
    │   ├── standing_approvals.yaml   # Standing approval schemas
    │   ├── shared.yaml          # Shared schemas (Action, ExecutionStatus, etc.)
    │   └── errors.yaml          # Error response schemas
    ├── paths/                   # API endpoints
    │   ├── actions.yaml         # POST /v1/actions/execute
    │   ├── action_config_templates.yaml # Action config template discovery
    │   ├── action_configurations.yaml   # Action configuration CRUD
    │   ├── agents.yaml          # Agent management endpoints
    │   ├── agent_connectors.yaml # Agent-connector management
    │   ├── approvals.yaml       # Approval lifecycle endpoints
    │   ├── audit_events.yaml    # Audit event listing and export
    │   ├── billing.yaml         # Billing plan, usage, upgrade, downgrade, invoices
    │   ├── capabilities.yaml    # GET /v1/agents/{id}/capabilities
    │   ├── config.yaml          # Server configuration
    │   ├── connectors.yaml      # GET /v1/connectors, GET /v1/connectors/{id}
    │   ├── credentials.yaml     # Credential vault endpoints
    │   ├── onboarding.yaml      # Onboarding workflow
    │   ├── profiles.yaml        # User profile management
    │   ├── push_subscriptions.yaml # Web Push subscription management
    │   ├── registration.yaml    # Agent registration endpoints
    │   ├── registration_invites.yaml # Registration invite creation
    │   └── standing_approvals.yaml   # Standing approval management
    ├── parameters/              # Reusable parameters
    │   └── common.yaml          # Path parameters (agent_id, approval_id, etc.)
    ├── responses/               # Reusable responses
    │   └── errors.yaml          # Standard error responses
    └── securitySchemes/         # Authentication schemes
        └── security.yaml        # Supabase session JWT and agent signature auth
```

## API Endpoint Groups

| Group | Endpoints | Auth | Description |
|---|---|---|---|
| **Connectors** | `GET /v1/connectors`, `GET /v1/connectors/{id}` | Public | Discover available integrations and actions |
| **Registration** | `POST /invite/{code}`, `POST /v1/agents/{id}/verify` | Signature | Agent identity setup via invite URL |
| **Agents** | `GET/PUT/DELETE /v1/agents/{id}`, `POST .../deactivate` | Session | Agent management |
| **Approvals** | `POST /v1/approvals/request`, `GET .../status`, `POST .../cancel`, `POST .../approve`, `POST .../deny` | Signature/Session | Approval lifecycle |
| **Actions** | `POST /v1/actions/execute` | Signature | Execute actions via standing approvals |
| **Credentials** | `POST/GET /v1/credentials`, `DELETE .../credentials/{id}` | Session | Manage stored API credentials |
| **Capabilities** | `GET /v1/agents/{id}/capabilities` | Signature | Agent capability discovery |
| **Standing Approvals** | `GET/POST /v1/standing-approvals`, `POST .../revoke`, `POST .../execute` | Session/Signature | Pre-authorized agent actions |
| **Audit Events** | `GET /v1/audit-events`, `GET /v1/audit-logs` | Session | Activity feed and audit trail |
| **Billing** | `GET /v1/billing/plan`, `GET .../usage`, `POST .../upgrade`, `POST .../downgrade`, `GET .../invoices` | Session | Subscription, usage, and invoices |
| **Profiles** | `GET/PUT /v1/profile`, `GET/PUT .../notification-preferences` | Session | User profile management |
| **Config** | `GET /v1/config` | Session | Server configuration and feature flags |

## Usage

### Viewing the Specification

#### Redoc (Recommended)

Interactive, clean documentation:

```bash
# Install Redoc CLI
npm install -g @redocly/cli

# Serve locally
redocly preview-docs spec/openapi/openapi.yaml
```

Open `http://localhost:8080` in your browser.

#### Swagger UI

Traditional API explorer with "Try it out" functionality:

```bash
# Install Swagger UI Watcher
npm install -g swagger-ui-watcher

# Serve locally
swagger-ui-watcher spec/openapi/openapi.yaml
```

Open `http://localhost:3000` in your browser.

### Validation

Ensure the specification is valid OpenAPI 3.0:

```bash
# Using Redocly
redocly lint spec/openapi/openapi.yaml
```

### Bundling

Create a single-file specification (useful for distribution):

```bash
# Using Redocly
redocly bundle spec/openapi/openapi.yaml -o spec/openapi/openapi.bundle.yaml
```

### Code Generation

Generate client libraries:

```bash
# Python client
docker run --rm -v ${PWD}:/local openapitools/openapi-generator-cli generate \
  -i /local/spec/openapi/openapi.yaml \
  -g python \
  -o /local/generated/python-client

# TypeScript/Axios client
docker run --rm -v ${PWD}:/local openapitools/openapi-generator-cli generate \
  -i /local/spec/openapi/openapi.yaml \
  -g typescript-axios \
  -o /local/generated/typescript-client

# Go client
docker run --rm -v ${PWD}:/local openapitools/openapi-generator-cli generate \
  -i /local/spec/openapi/openapi.yaml \
  -g go \
  -o /local/generated/go-client
```

## Versioning

The `info.version` in the OpenAPI spec tracks the **product/API version** using [semver](https://semver.org/). While pre-launch, the version is `0.x.y` to signal that breaking changes may occur without a major version bump.

The `/v1/` prefix in URL paths (e.g., `/api/v1/approvals/request`) is a **routing convention**, not a semantic version. It simply means "first version of the API surface." There are no plans to introduce `/v2/` or change this prefix — if the API evolves in breaking ways, we'll use the spec version, feature flags, or request headers to manage compatibility rather than minting a new URL prefix.

## Key Features

### Strict Validation

All schemas include strict validation rules:

- **maxLength**: Prevents unbounded strings
- **pattern**: Validates formats (UUIDs, agent IDs, confirmation codes)
- **enum**: Restricts values to known constants
- **minLength/maxLength**: Enforces reasonable bounds
- **JSON Schema**: Full JSON Schema Draft 7 support

### Comprehensive Examples

Every endpoint includes multiple examples:

- **Basic usage**: Minimal valid requests/responses
- **Full features**: Requests with optional fields
- **Error cases**: All error codes with detailed examples
- **Edge cases**: Boundary conditions and special scenarios

### Security Schemes

Two authentication methods are defined:

1. **PermissionSlipSignature**: Request signature using agent's private key (for agent-facing endpoints)
2. **SupabaseSession**: JWT bearer token from Supabase Auth (for user-facing dashboard endpoints)

### Error Handling

Standardized error responses for all endpoints:

- Machine-readable error codes
- Human-readable messages
- Retryability indicators
- Optional error details
- Trace IDs for debugging

## Related Documentation

- **[API Specification](../../docs/spec/api.md)**: Complete HTTP API reference
- **[Authentication Specification](../../docs/spec/authentication.md)**: Cryptographic details for key generation and signatures
- **[Notifications Specification](../../docs/spec/notifications.md)**: How users are notified of pending approvals
- **[Terminology](../../docs/spec/terminology.md)**: Definitions of core concepts

## Editing the Specification

When updating the specification:

1. **Edit modular files**: Update individual schema/path files, not the main `openapi.yaml`
2. **Validate changes**: Run `redocly lint` before committing
3. **Regenerate bundle**: Run `redocly bundle` to update `openapi.bundle.yaml`
4. **Update examples**: Ensure examples match schema changes
5. **Update this README**: Document any structural changes

## Tools

Recommended tools for working with OpenAPI specs:

- **[Redocly CLI](https://redocly.com/docs/cli/)**: Validation, bundling, and preview
- **[Swagger Editor](https://editor.swagger.io/)**: Online editor with live preview
- **[OpenAPI Generator](https://openapi-generator.tech/)**: Code generation for 50+ languages
- **[Stoplight Studio](https://stoplight.io/studio)**: Visual OpenAPI editor
- **[Spectral](https://stoplight.io/open-source/spectral)**: Advanced linting and style enforcement

