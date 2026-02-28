# Permission Slip — OpenAPI 3.0 Specification

This directory contains the complete OpenAPI 3.0 specification for the Permission Slip API.

## Overview

Permission Slip is a centralized middle-man service that sits between AI agents and external APIs. Agents submit pre-defined **actions** for human approval, and Permission Slip executes them using the user's stored credentials. Actions are the core primitive — structured templates (like `email.send`, `flight.book`, `payment.charge`) that define what an agent can request, with validated parameters and human-readable display formats.

This specification defines the HTTP API that agents use to interact with Permission Slip at `https://app.permissionslip.dev`. Agents integrate with this single endpoint for everything — action discovery, registration, approval, and execution.

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
    │   ├── credentials.yaml     # Credential vault schemas
    │   ├── connectors.yaml      # Connector/integration schemas
    │   ├── capabilities.yaml    # Agent capability discovery schemas
    │   ├── shared.yaml          # Shared schemas (Action, Token, etc.)
    │   └── errors.yaml          # Error response schemas
    ├── paths/                   # API endpoints
    │   ├── connectors.yaml      # GET /v1/connectors, GET /v1/connectors/{id}
    │   ├── registration.yaml    # Agent registration endpoints
    │   ├── approvals.yaml       # Approval lifecycle endpoints
    │   ├── actions.yaml         # POST /v1/actions/execute
    │   ├── credentials.yaml     # Credential vault endpoints
    │   ├── capabilities.yaml    # GET /v1/agents/{id}/capabilities
    │   └── standing_approvals.yaml # Standing approval discovery
    ├── parameters/              # Reusable parameters
    │   └── common.yaml          # Path parameters (agent_id, approval_id, etc.)
    ├── responses/               # Reusable responses
    │   └── errors.yaml          # Standard error responses
    └── securitySchemes/         # Authentication schemes
        └── security.yaml        # Signature-based and JWT auth
```

## API Endpoint Groups

| Group | Endpoints | Description |
|---|---|---|
| **Connectors** | `GET /v1/connectors`, `GET /v1/connectors/{id}` | Discover available integrations and actions |
| **Registration** | `POST /invite/{invite_code}`, `POST /v1/agents/{id}/verify` | Agent identity setup via invite URL |
| **Approvals** | `POST /v1/approvals/request`, `POST .../verify`, `POST .../cancel` | Approval lifecycle |
| **Actions** | `POST /v1/actions/execute` | Execute approved actions |
| **Credentials** | `POST /v1/credentials`, `GET /v1/credentials`, `DELETE /v1/credentials/{id}` | Manage stored API credentials |
| **Capabilities** | `GET /v1/agents/{id}/capabilities` | Agent capability discovery (connectors, actions, schemas, authorization) |
| **Standing Approvals** | `GET /v1/agents/{id}/standing-approvals` | Discover active standing approvals |

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

1. **PermissionSlipSignature**: Request signature using agent's private key (for all endpoints except connector discovery)
2. **BearerToken**: JWT bearer token (issued after approval verification, used in action execution)

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

