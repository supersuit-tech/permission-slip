# DocuSign Connector

E-signature integration using the [DocuSign eSignature REST API v2.1](https://developers.docusign.com/docs/esign-rest-api/).

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `access_token` | Yes | DocuSign OAuth 2.0 access token |
| `account_id` | Yes | DocuSign account ID (found in account settings) |
| `base_url` | No | API base URL — defaults to demo sandbox (`https://demo.docusign.net/restapi/v2.1`). Set to `https://na1.docusign.net/restapi/v2.1` for production (US), or the appropriate regional URL. |

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

## Production setup

The connector defaults to the DocuSign developer sandbox. For production:

1. Set `base_url` to your production API URL (e.g., `https://na1.docusign.net/restapi/v2.1`)
2. Use a production OAuth access token
3. Ensure your DocuSign account has the necessary API permissions

Regional base URLs:
- US: `https://na1.docusign.net/restapi/v2.1` through `https://na4.docusign.net/restapi/v2.1`
- EU: `https://eu.docusign.net/restapi/v2.1`
- AU: `https://au.docusign.net/restapi/v2.1`
