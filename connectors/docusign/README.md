# DocuSign Connector

E-signature integration using the [DocuSign eSignature REST API v2.1](https://developers.docusign.com/docs/esign-rest-api/).

## Authentication

The connector supports two authentication methods:

### OAuth 2.0 (Recommended)

Connect via **Settings → Connected Accounts → DocuSign**. The platform handles the full OAuth authorization code flow:

1. User clicks **Connect** and authorizes the app on DocuSign's consent screen
2. After authorization, the platform calls DocuSign's userinfo endpoint to retrieve the user's default account ID and regional API base URL
3. All three values (`access_token`, `account_id`, `base_url`) are stored securely and used automatically — no manual credential entry required

**Requires:** `DOCUSIGN_CLIENT_ID` and `DOCUSIGN_CLIENT_SECRET` environment variables. See [DocuSign OAuth Setup](../../docs/oauth-setup.md#docusign-oauth-setup) for instructions.

### RSA Key / Custom Auth

For service-to-service integrations or users who prefer manual setup, you can store credentials directly:

| Key | Required | Description |
|-----|----------|-------------|
| `access_token` | Yes | DocuSign OAuth 2.0 access token |
| `account_id` | Yes | DocuSign account ID (found in account settings) |
| `base_url` | No | API base URL — defaults to demo sandbox (`https://demo.docusign.net/restapi/v2.1`). Set to `https://na1.docusign.net/restapi/v2.1` for production (US), or the appropriate regional URL. Must be a `*.docusign.net` HTTPS URL (other domains are rejected to prevent SSRF). |

## Actions

| Action | Risk | Description |
|--------|------|-------------|
| `docusign.create_envelope` | medium | Create a draft envelope from a template with recipients |
| `docusign.send_envelope` | **high** | Send a draft envelope for signature (legally binding) |
| `docusign.check_status` | low | Check envelope status with per-recipient signing progress |
| `docusign.download_signed` | low | Download signed documents as base64-encoded PDF |
| `docusign.list_templates` | low | Browse available templates with search and pagination |
| `docusign.void_envelope` | **high** | Cancel an in-progress signing |
| `docusign.update_recipients` | medium | Add or update signers on a draft envelope |

## Typical workflow

1. **List templates** — find the right template for your document
2. **Create envelope** — create a draft from a template with recipients (medium risk, requires approval)
3. **Send envelope** — send for signature (high risk, requires separate approval)
4. **Check status** — monitor signing progress and see who has/hasn't signed
5. **Download signed** — retrieve the completed PDF for record-keeping

The two-step create → send flow is intentional: it separates document preparation (medium risk) from sending legally binding documents to external parties (high risk), giving humans a clear approval checkpoint before anything goes out.

## Error handling

The connector maps DocuSign API errors to typed connector errors:

| DocuSign error | Connector error type | Notes |
|----------------|---------------------|-------|
| `AUTHORIZATION_INVALID_TOKEN` | `AuthError` | Token expired or invalid |
| `ENVELOPE_DOES_NOT_EXIST` | `ValidationError` | Incorrect envelope_id |
| `TEMPLATE_NOT_FOUND` | `ValidationError` | Incorrect template_id |
| `ENVELOPE_NOT_IN_CORRECT_STATE` | `ExternalError` | e.g. sending an already-sent envelope |
| `INVALID_EMAIL_ADDRESS_FOR_RECIPIENT` | `ValidationError` | Bad email in recipient list |
| HTTP 429 | `RateLimitError` | Includes Retry-After from DocuSign |
| HTTP 404 (no error code) | `ValidationError` | Resource not found |

Error messages include actionable suggestions (e.g. "Use docusign.list_templates to browse available templates").

## Security

- **URL path escaping**: All user-provided values (envelope IDs, document IDs, account IDs) are escaped with `url.PathEscape` before being interpolated into API paths to prevent path traversal.
- **SSRF prevention**: The `base_url` credential is validated against an allowlist — only `*.docusign.net` HTTPS URLs are accepted. This prevents credential exfiltration via a malicious base URL.
- **Response size limits**: API responses are capped at 50 MB (`io.LimitReader`) to prevent memory exhaustion from unexpectedly large responses.
- **Error body truncation**: Response bodies in error messages are truncated to 512 bytes to prevent log bloat and potential information leakage.

## Production setup

The connector defaults to the DocuSign developer sandbox. For production:

1. Set `base_url` to your production API URL (e.g., `https://na1.docusign.net/restapi/v2.1`)
2. Use a production OAuth access token
3. Ensure your DocuSign account has the necessary API permissions

Regional base URLs:
- US: `https://na1.docusign.net/restapi/v2.1` through `https://na4.docusign.net/restapi/v2.1`
- EU: `https://eu.docusign.net/restapi/v2.1`
- AU: `https://au.docusign.net/restapi/v2.1`

## File organization

| File | Purpose |
|------|---------|
| `docusign.go` | Core connector struct, constructor, credential helpers, `parseParams` DRY helper |
| `docusign_manifest.go` | `Manifest()` — action schemas, credential config, templates for DB seeding |
| `docusign_http.go` | HTTP transport: `doJSON`, `doRaw`, base URL resolution and SSRF validation |
| `docusign_errors.go` | Error mapping from DocuSign API codes to typed connector errors |
| `create_envelope.go` | `docusign.create_envelope` action |
| `send_envelope.go` | `docusign.send_envelope` action |
| `check_status.go` | `docusign.check_status` action (with optional recipient details) |
| `download_signed.go` | `docusign.download_signed` action (PDF download as base64) |
| `list_templates.go` | `docusign.list_templates` action (search + pagination) |
| `void_envelope.go` | `docusign.void_envelope` action |
| `update_recipients.go` | `docusign.update_recipients` action |
