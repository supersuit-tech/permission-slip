# Tresorit connector

Permission Slip integrates with [Tresorit's S3-compatible API](https://support.tresorit.com/hc/en-us/articles/33971802915858-Tresorit-API) through a **local gateway** that runs in your environment (Docker). End-to-end encryption happens inside that gateway — Permission Slip is a SigV4-signed S3 client against your loopback endpoint.

This mirrors the [Proton Mail connector](../protonmail/) pattern: local proxy, custom credentials, and a live reachability check when credentials are saved.

## Prerequisites

1. **Tresorit API access** — currently Early Adopter / waitlist-gated. Email support@tresorit.com to request access.
2. **Docker** — run the gateway from [tresorit/s3-api](https://github.com/tresorit/s3-api).
3. **Permission Slip on the same host** — the gateway listens on loopback (default `http://127.0.0.1:3000`).

## Gateway setup

1. Clone and configure the Tresorit S3 API deployment per the upstream README.
2. Log in so `credentials.json` is generated with `client_id` and `client_secret`.
3. Start the gateway (`docker compose up` or equivalent).
4. Confirm it responds:

```bash
AWS_ACCESS_KEY_ID="<client_id>" \
AWS_SECRET_ACCESS_KEY="<client_secret>" \
aws s3api list-buckets --endpoint-url http://127.0.0.1:3000 --region us-east-1
```

Tresorit requires **path-style** addressing and a fixed region of `us-east-1`.

## Credentials

| Field | Description |
|-------|-------------|
| `access_key` | `client_id` from the gateway's `credentials.json` |
| `secret_key` | `client_secret` from `credentials.json` (stored as a secret) |
| `endpoint_url` | Gateway URL, e.g. `http://127.0.0.1:3000` |

Saving credentials runs a live `ListBuckets` against the gateway. If the container is not running or the URL is wrong, validation fails with an actionable error.

## Actions

| Action | Risk | Description |
|--------|------|-------------|
| `tresorit.list_files` | low | List files/folders in a Tresor (`tresor` = bucket name), optional `prefix` |
| `tresorit.download_file` | medium | Download a file; content returned as base64 |
| `tresorit.upload_file` | medium | Upload a file from base64 `content` |
| `tresorit.create_folder` | medium | Create a folder (S3 prefix marker) |
| `tresorit.delete_file` | high | Permanently delete a file |

### Parameters

- **tresor** — Tresor name as exposed by the gateway (S3 bucket).
- **key** / **path** — Object key within the Tresor. Folders use a trailing `/`.
- **content** — Base64-encoded bytes for uploads (same convention as Dropbox).

## Architecture notes

- Signing reuses the shared `connectors/s3sigv4` helper (also used by the AWS connector for SigV4).
- No OAuth — custom credentials only.
- HTTP is supported for loopback gateways; terminate TLS locally if required.

## Troubleshooting

| Symptom | Likely cause |
|---------|----------------|
| Connection refused on credential save | Gateway not running or wrong `endpoint_url` |
| 403 SignatureDoesNotMatch | Wrong `access_key` / `secret_key` |
| 404 on list/download | Wrong Tresor name or file path |
