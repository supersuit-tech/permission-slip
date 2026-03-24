# Amazon DynamoDB connector

Built-in actions: `dynamodb.get_item`, `dynamodb.put_item`, `dynamodb.delete_item`, `dynamodb.query`.

## Credentials (vault)

| Key | Required | Description |
|-----|----------|-------------|
| `access_key_id` | yes | IAM access key ID |
| `secret_access_key` | yes | IAM secret access key |
| `session_token` | no | Temporary credentials (STS) |
| `endpoint_url` | no | Custom API endpoint (e.g. `http://localhost:4566` for LocalStack) |

Use the same key names as the **AWS** connector so users can reuse IAM user keys. For LocalStack or other emulators, set `endpoint_url` to the DynamoDB endpoint base URL (no trailing slash).

## Safety parameters

Every action requires **`allowed_tables`**: the requested `table` must appear in that list. Optional **`allowed_read_attributes`** / **`allowed_write_attributes`** further restrict returned or written attributes. Query **`limit`** is capped at 1000 (default 100).
