# AWS Connector

The AWS connector integrates Permission Slip with the [AWS REST APIs](https://docs.aws.amazon.com/general/latest/gr/aws-apis.html) for EC2, S3, CloudWatch, and RDS. It uses plain `net/http` with [AWS Signature V4](https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_aws-signing.html) authentication — no AWS SDK dependency.

## Connector ID

`aws`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `access_key_id` | Yes | AWS access key ID (starts with `AKIA` for long-term keys or `ASIA` for temporary credentials) |
| `secret_access_key` | Yes | AWS secret access key paired with the access key ID |
| `session_token` | No | Session token for temporary credentials (from STS AssumeRole, etc.) |

The credential `auth_type` in the database is `custom`. Credentials are stored encrypted in Supabase Vault and decrypted only at execution time.

To create an access key, see [Managing access keys for IAM users](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html). Use an IAM user or role with least-privilege permissions for only the actions you need.

## Actions

### `aws.describe_instances`

Lists and describes EC2 instances with their current status, IP addresses, tags, and availability zones.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `region` | string | Yes | AWS region (e.g. `us-east-1`) |
| `instance_ids` | array of strings | No | Instance IDs to filter by (e.g. `["i-1234567890abcdef0"]`) |

**Response:**

```json
{
  "instances": [
    {
      "instance_id": "i-1234567890abcdef0",
      "instance_type": "t2.micro",
      "state": "running",
      "private_ip": "10.0.0.1",
      "public_ip": "54.1.2.3",
      "availability_zone": "us-east-1a",
      "launch_time": "2024-01-01T00:00:00Z",
      "tags": {"Name": "my-instance"}
    }
  ],
  "count": 1,
  "region": "us-east-1"
}
```

**AWS API:** `POST` EC2 `DescribeInstances` ([docs](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeInstances.html))

**Required IAM permissions:** `ec2:DescribeInstances`

---

### `aws.start_instance`

Starts a stopped EC2 instance.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `region` | string | Yes | AWS region (e.g. `us-east-1`) |
| `instance_id` | string | Yes | EC2 instance ID (must start with `i-`) |

**Response:**

```json
{
  "instance_id": "i-1234567890abcdef0",
  "current_state": "pending",
  "previous_state": "stopped"
}
```

**AWS API:** `POST` EC2 `StartInstances` ([docs](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_StartInstances.html))

**Required IAM permissions:** `ec2:StartInstances`

---

### `aws.stop_instance`

Stops a running EC2 instance.

**Risk level:** high

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `region` | string | Yes | AWS region (e.g. `us-east-1`) |
| `instance_id` | string | Yes | EC2 instance ID (must start with `i-`) |

**Response:**

```json
{
  "instance_id": "i-1234567890abcdef0",
  "current_state": "stopping",
  "previous_state": "running"
}
```

**AWS API:** `POST` EC2 `StopInstances` ([docs](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_StopInstances.html))

**Required IAM permissions:** `ec2:StopInstances`

---

### `aws.restart_instance`

Reboots a running EC2 instance.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `region` | string | Yes | AWS region (e.g. `us-east-1`) |
| `instance_id` | string | Yes | EC2 instance ID (must start with `i-`) |

**Response:**

```json
{
  "instance_id": "i-1234567890abcdef0",
  "status": "rebooting"
}
```

**AWS API:** `POST` EC2 `RebootInstances` ([docs](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_RebootInstances.html))

**Required IAM permissions:** `ec2:RebootInstances`

---

### `aws.get_metrics`

Retrieves CloudWatch metrics with configurable dimensions, statistics, and time ranges.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `region` | string | Yes | — | AWS region (e.g. `us-east-1`) |
| `namespace` | string | Yes | — | CloudWatch namespace (e.g. `AWS/EC2`, `AWS/RDS`) |
| `metric_name` | string | Yes | — | Metric name (e.g. `CPUUtilization`) |
| `start_time` | string | Yes | — | Start time in RFC 3339 format (e.g. `2024-01-15T00:00:00Z`) |
| `end_time` | string | Yes | — | End time in RFC 3339 format |
| `period` | integer | Yes | — | Aggregation period in seconds (e.g. 60, 300, 3600) |
| `stat` | string | No | `"Average"` | One of `Average`, `Sum`, `Minimum`, `Maximum`, `SampleCount` |
| `dimensions` | array | No | — | Array of `{name, value}` dimension filters |

**Response:**

```json
{
  "label": "CPUUtilization",
  "datapoints": [
    {"timestamp": "2024-01-01T00:00:00Z", "value": 25.5, "unit": "Percent"},
    {"timestamp": "2024-01-01T00:05:00Z", "value": 30.2, "unit": "Percent"}
  ],
  "stat": "Average",
  "region": "us-east-1"
}
```

**AWS API:** `POST` CloudWatch `GetMetricStatistics` ([docs](https://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/API_GetMetricStatistics.html))

**Required IAM permissions:** `cloudwatch:GetMetricStatistics`

---

### `aws.list_s3_objects`

Lists objects in an S3 bucket with optional prefix filtering.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `region` | string | Yes | — | AWS region (e.g. `us-east-1`) |
| `bucket` | string | Yes | — | S3 bucket name |
| `prefix` | string | No | — | Object key prefix to filter by |
| `max_keys` | integer | No | `1000` | Maximum number of keys to return (1–1000) |

**Response:**

```json
{
  "bucket": "my-bucket",
  "prefix": "logs/",
  "objects": [
    {"key": "logs/2024-01-01.log", "last_modified": "2024-01-01T12:00:00Z", "size": 1024, "storage_class": "STANDARD"}
  ],
  "count": 1,
  "is_truncated": false
}
```

**AWS API:** `GET` S3 `ListObjectsV2` ([docs](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListObjectsV2.html))

**Required IAM permissions:** `s3:ListBucket`

---

### `aws.create_presigned_url`

Generates a time-limited presigned URL for S3 object upload or download. This action does **not** make an API call — it constructs the signed URL locally using the provided credentials.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `region` | string | Yes | — | AWS region (e.g. `us-east-1`) |
| `bucket` | string | Yes | — | S3 bucket name |
| `key` | string | Yes | — | S3 object key |
| `operation` | string | Yes | — | `GET` for download, `PUT` for upload |
| `expires_in` | integer | No | `3600` | URL expiration time in seconds (1–604800, max 7 days) |

**Response:**

```json
{
  "url": "https://s3.us-east-1.amazonaws.com/my-bucket/file.txt?X-Amz-Algorithm=...",
  "operation": "GET",
  "bucket": "my-bucket",
  "key": "file.txt",
  "expires_in": 3600
}
```

**Required IAM permissions:** `s3:GetObject` (for GET) or `s3:PutObject` (for PUT)

---

### `aws.describe_rds_instances`

Lists and describes RDS database instances and their status.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `region` | string | Yes | AWS region (e.g. `us-east-1`) |
| `db_instance_id` | string | No | Specific DB instance identifier to describe |

**Response:**

```json
{
  "instances": [
    {
      "db_instance_id": "my-database",
      "db_instance_class": "db.t3.micro",
      "engine": "postgres",
      "engine_version": "15.4",
      "status": "available",
      "allocated_storage_gb": 20,
      "availability_zone": "us-east-1a",
      "multi_az": false,
      "storage_type": "gp3",
      "endpoint_address": "my-database.abc123.us-east-1.rds.amazonaws.com",
      "endpoint_port": 5432,
      "publicly_accessible": false,
      "storage_encrypted": true,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "count": 1,
  "region": "us-east-1"
}
```

**AWS API:** `POST` RDS `DescribeDBInstances` ([docs](https://docs.aws.amazon.com/AmazonRDS/latest/APIReference/API_DescribeDBInstances.html))

**Required IAM permissions:** `rds:DescribeDBInstances`

## Error Handling

The connector maps AWS API responses to typed connector errors:

| AWS Status | Connector Error | HTTP Response | Recovery Hint |
|------------|----------------|---------------|---------------|
| 401 | `AuthError` | 502 Bad Gateway | Check that credentials are valid and not expired |
| 403 | `AuthError` | 502 Bad Gateway | Verify IAM user/role has required permissions |
| 400 | `ValidationError` | 400 Bad Request | Check parameter values against the action schema |
| 404 | `ValidationError` | 400 Bad Request | Verify resource ID, region, and that the resource exists |
| 429 | `RateLimitError` | 429 Too Many Requests | Retry after a brief delay |
| Other 4xx/5xx | `ExternalError` | 502 Bad Gateway | — |
| Client timeout | `TimeoutError` | 504 Gateway Timeout | — |

## Adding a New Action

Each action lives in its own file. To add one:

1. Create `connectors/aws/<action>.go` with a params struct, `validate()`, and an `Execute` method.
2. Use `parseAndValidate[myParams](req.Parameters)` to handle JSON unmarshaling and validation in one call.
3. Use shared validators: `validateRegion()`, `validateInstanceID()`, `validateRFC3339()`.
4. Call `a.conn.do(ctx, creds, method, host, path, query, body)` for signed HTTP requests — it handles Signature V4 signing, response checking, and error mapping.
5. Return `connectors.JSONResult(response)` to wrap the result.
6. Register the action in `Actions()` inside `aws.go`.
7. Add the action to the `Manifest()` return value with a `ParametersSchema` (include `default`, `minimum`, `maximum` where applicable).
8. Add one or more `ManifestTemplate` entries for common permission presets.
9. Add tests in `<action>_test.go` using `httptest.NewTLSServer` and `testTransport`.

## File Structure

```
connectors/aws/
├── aws.go                         # AWSConnector struct, New(), Manifest(), SignV4, do()
├── validation.go                  # Shared validation: parseAndValidate, validateRegion, etc.
├── response.go                    # AWS XML error parsing and typed error mapping
├── describe_instances.go          # aws.describe_instances + shared instanceIDParams
├── start_instance.go              # aws.start_instance
├── stop_instance.go               # aws.stop_instance
├── restart_instance.go            # aws.restart_instance
├── get_metrics.go                 # aws.get_metrics (CloudWatch)
├── list_s3_objects.go             # aws.list_s3_objects
├── create_presigned_url.go        # aws.create_presigned_url (local signing, no API call)
├── describe_rds_instances.go      # aws.describe_rds_instances
├── helpers_test.go                # Shared test helpers (validCreds, testTransport)
├── aws_test.go                    # Connector-level tests
├── describe_instances_test.go     # Tests for aws.describe_instances
├── start_instance_test.go         # Tests for start/stop/restart
├── get_metrics_test.go            # Tests for aws.get_metrics
├── list_s3_objects_test.go        # Tests for aws.list_s3_objects
├── create_presigned_url_test.go   # Tests for aws.create_presigned_url
├── describe_rds_instances_test.go # Tests for aws.describe_rds_instances
├── response_test.go               # Tests for error response parsing
└── README.md                      # This file
```

## Testing

All tests use `httptest.NewTLSServer` with a `testTransport` that redirects AWS API calls to the test server — no real AWS API calls are made.

```bash
go test ./connectors/aws/... -v
```
