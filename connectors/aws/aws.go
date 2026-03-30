// Package aws implements the AWS connector for the Permission Slip connector
// execution layer. It uses plain net/http calls against the AWS REST APIs,
// authenticating with AWS Signature V4. This keeps the dependency footprint
// minimal (no AWS SDK).
package aws

import (
	_ "embed"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const defaultTimeout = 30 * time.Second

// maxResponseBytes caps the response body size to prevent memory exhaustion
// from oversized or malicious upstream responses.
const maxResponseBytes = 10 * 1024 * 1024 // 10 MB

// AWSConnector owns the shared HTTP client used by all AWS actions.
// Actions hold a pointer back to the connector to access the client.
type AWSConnector struct {
	client *http.Client
}

// New creates an AWSConnector with sensible defaults (30s timeout).
func New() *AWSConnector {
	return &AWSConnector{
		client: &http.Client{Timeout: defaultTimeout},
	}
}

// newForTest creates an AWSConnector that uses a custom HTTP client.
func newForTest(client *http.Client) *AWSConnector {
	return &AWSConnector{
		client: client,
	}
}

// ID returns "aws", matching the connectors.id in the database.
func (c *AWSConnector) ID() string { return "aws" }

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup, replacing manual seed.go files.
//go:embed logo.svg
var logoSVG string

func (c *AWSConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "aws",
		Name:        "AWS",
		Description: "Amazon Web Services integration for cloud infrastructure management",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "aws.describe_instances",
				Name:        "Describe EC2 Instances",
				Description: "List and describe EC2 instances with their current status",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["region"],
					"properties": {
						"region": {
							"type": "string",
							"description": "AWS region (e.g. us-east-1)"
						},
						"instance_ids": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Optional list of instance IDs to filter by"
						}
					}
				}`)),
			},
			{
				ActionType:  "aws.start_instance",
				Name:        "Start EC2 Instance",
				Description: "Start a stopped EC2 instance",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["region", "instance_id"],
					"properties": {
						"region": {
							"type": "string",
							"description": "AWS region (e.g. us-east-1)"
						},
						"instance_id": {
							"type": "string",
							"description": "EC2 instance ID (e.g. i-1234567890abcdef0)"
						}
					}
				}`)),
			},
			{
				ActionType:  "aws.stop_instance",
				Name:        "Stop EC2 Instance",
				Description: "Stop a running EC2 instance",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["region", "instance_id"],
					"properties": {
						"region": {
							"type": "string",
							"description": "AWS region (e.g. us-east-1)"
						},
						"instance_id": {
							"type": "string",
							"description": "EC2 instance ID (e.g. i-1234567890abcdef0)"
						}
					}
				}`)),
			},
			{
				ActionType:  "aws.restart_instance",
				Name:        "Restart EC2 Instance",
				Description: "Reboot a running EC2 instance",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["region", "instance_id"],
					"properties": {
						"region": {
							"type": "string",
							"description": "AWS region (e.g. us-east-1)"
						},
						"instance_id": {
							"type": "string",
							"description": "EC2 instance ID (e.g. i-1234567890abcdef0)"
						}
					}
				}`)),
			},
			{
				ActionType:  "aws.get_metrics",
				Name:        "Get CloudWatch Metrics",
				Description: "Retrieve CloudWatch metrics (CPU, memory, custom metrics)",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["region", "namespace", "metric_name", "start_time", "end_time", "period"],
					"properties": {
						"region": {
							"type": "string",
							"description": "AWS region (e.g. us-east-1)"
						},
						"namespace": {
							"type": "string",
							"description": "CloudWatch namespace (e.g. AWS/EC2, AWS/RDS)"
						},
						"metric_name": {
							"type": "string",
							"description": "Metric name (e.g. CPUUtilization, FreeableMemory)"
						},
						"dimensions": {
							"type": "array",
							"items": {
								"type": "object",
								"required": ["name", "value"],
								"properties": {
									"name": {"type": "string", "description": "Dimension name (e.g. InstanceId)"},
									"value": {"type": "string", "description": "Dimension value (e.g. i-1234567890abcdef0)"}
								}
							},
							"description": "CloudWatch dimensions to filter by"
						},
						"start_time": {
							"type": "string",
							"format": "date-time",
							"description": "Start time in RFC 3339 format (e.g. 2024-01-15T00:00:00Z)"
						},
						"end_time": {
							"type": "string",
							"format": "date-time",
							"description": "End time in RFC 3339 format (e.g. 2024-01-15T01:00:00Z)"
						},
						"period": {
							"type": "integer",
							"minimum": 1,
							"description": "Aggregation period in seconds (e.g. 60, 300, 3600)"
						},
						"stat": {
							"type": "string",
							"default": "Average",
							"enum": ["Average", "Sum", "Minimum", "Maximum", "SampleCount"],
							"description": "Statistic to retrieve"
						}
					}
				}`)),
			},
			{
				ActionType:  "aws.list_s3_objects",
				Name:        "List S3 Objects",
				Description: "List objects in an S3 bucket with optional prefix filter",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["region", "bucket"],
					"properties": {
						"region": {
							"type": "string",
							"description": "AWS region (e.g. us-east-1)"
						},
						"bucket": {
							"type": "string",
							"description": "S3 bucket name"
						},
						"prefix": {
							"type": "string",
							"description": "Object key prefix to filter by"
						},
						"max_keys": {
							"type": "integer",
							"default": 1000,
							"minimum": 1,
							"maximum": 1000,
							"description": "Maximum number of keys to return"
						}
					}
				}`)),
			},
			{
				ActionType:  "aws.create_presigned_url",
				Name:        "Create S3 Presigned URL",
				Description: "Generate a time-limited presigned URL for S3 object upload or download",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["region", "bucket", "key", "operation"],
					"properties": {
						"region": {
							"type": "string",
							"description": "AWS region (e.g. us-east-1)"
						},
						"bucket": {
							"type": "string",
							"description": "S3 bucket name"
						},
						"key": {
							"type": "string",
							"description": "S3 object key"
						},
						"operation": {
							"type": "string",
							"enum": ["GET", "PUT"],
							"description": "Operation type: GET for download, PUT for upload"
						},
						"expires_in": {
							"type": "integer",
							"default": 3600,
							"minimum": 1,
							"maximum": 604800,
							"description": "URL expiration time in seconds (max 7 days)"
						}
					}
				}`)),
			},
			{
				ActionType:  "aws.describe_rds_instances",
				Name:        "Describe RDS Instances",
				Description: "List and describe RDS database instances and their status",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["region"],
					"properties": {
						"region": {
							"type": "string",
							"description": "AWS region (e.g. us-east-1)"
						},
						"db_instance_id": {
							"type": "string",
							"description": "Optional specific DB instance identifier to describe"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "aws",
				AuthType:        "custom",
				InstructionsURL: "https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_aws_read_only",
				ActionType:  "aws.describe_instances",
				Name:        "Read-only EC2 access",
				Description: "Agent can describe instances in any region.",
				Parameters:  json.RawMessage(`{"region":"*","instance_ids":"*"}`),
			},
			{
				ID:          "tpl_aws_restart_staging",
				ActionType:  "aws.restart_instance",
				Name:        "Restart staging instances",
				Description: "Agent can restart instances in a specific region. Instance ID is agent-controlled.",
				Parameters:  json.RawMessage(`{"region":"us-east-1","instance_id":"*"}`),
			},
			{
				ID:          "tpl_aws_metrics_read",
				ActionType:  "aws.get_metrics",
				Name:        "Read CloudWatch metrics",
				Description: "Agent can read any CloudWatch metrics. All fields are agent-controlled.",
				Parameters:  json.RawMessage(`{"region":"*","namespace":"*","metric_name":"*","start_time":"*","end_time":"*","period":"*","stat":"*","dimensions":"*"}`),
			},
			{
				ID:          "tpl_aws_s3_list",
				ActionType:  "aws.list_s3_objects",
				Name:        "List S3 objects in any bucket",
				Description: "Agent can list objects in any S3 bucket and region.",
				Parameters:  json.RawMessage(`{"region":"*","bucket":"*","prefix":"*","max_keys":"*"}`),
			},
			{
				ID:          "tpl_aws_s3_presigned",
				ActionType:  "aws.create_presigned_url",
				Name:        "Create presigned URLs",
				Description: "Agent can create presigned URLs for any S3 object.",
				Parameters:  json.RawMessage(`{"region":"*","bucket":"*","key":"*","operation":"*","expires_in":"*"}`),
			},
			{
				ID:          "tpl_aws_rds_describe",
				ActionType:  "aws.describe_rds_instances",
				Name:        "Read-only RDS access",
				Description: "Agent can describe RDS instances in any region.",
				Parameters:  json.RawMessage(`{"region":"*","db_instance_id":"*"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *AWSConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"aws.describe_instances":   &describeInstancesAction{conn: c},
		"aws.start_instance":       &startInstanceAction{conn: c},
		"aws.stop_instance":        &stopInstanceAction{conn: c},
		"aws.restart_instance":     &restartInstanceAction{conn: c},
		"aws.get_metrics":          &getMetricsAction{conn: c},
		"aws.list_s3_objects":      &listS3ObjectsAction{conn: c},
		"aws.create_presigned_url": &createPresignedURLAction{conn: c},
		"aws.describe_rds_instances": &describeRDSInstancesAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain the
// required AWS access key ID and secret access key.
func (c *AWSConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	accessKey, ok := creds.Get("access_key_id")
	if !ok || accessKey == "" {
		return &connectors.ValidationError{Message: "missing required credential: access_key_id"}
	}
	secretKey, ok := creds.Get("secret_access_key")
	if !ok || secretKey == "" {
		return &connectors.ValidationError{Message: "missing required credential: secret_access_key"}
	}
	return nil
}

// do sends a signed AWS API request. It handles AWS Signature V4 signing,
// sends the request, checks the response status, and returns the response body.
func (c *AWSConnector) do(ctx context.Context, creds connectors.Credentials, method, serviceHost, path, query string, body []byte) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	fullURL := "https://" + serviceHost + path
	if query != "" {
		fullURL += "?" + query
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("creating request: %v", err)}
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	}

	// Sign the request with AWS Signature V4.
	if err := c.signV4(req, creds, body); err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("AWS API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return nil, &connectors.CanceledError{Message: "AWS API request canceled"}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("AWS API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, respBytes); err != nil {
		return nil, err
	}

	return respBytes, nil
}

// signV4 signs an HTTP request using AWS Signature Version 4.
func (c *AWSConnector) signV4(req *http.Request, creds connectors.Credentials, payload []byte) error {
	accessKey, _ := creds.Get("access_key_id")
	secretKey, _ := creds.Get("secret_access_key")

	now := time.Now().UTC()
	datestamp := now.Format("20060102")
	amzdate := now.Format("20060102T150405Z")

	// Extract service and region from the host.
	// Host format: <service>.<region>.amazonaws.com
	service, region := parseServiceHost(req.Host)

	req.Header.Set("X-Amz-Date", amzdate)
	req.Header.Set("Host", req.Host)

	// Add security token header if session token is present.
	sessionToken, hasToken := creds.Get("session_token")
	if hasToken && sessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", sessionToken)
	}

	// Compute payload hash and set X-Amz-Content-Sha256 header.
	// S3 requires this header for Signature V4 Authorization-header requests.
	payloadHash := sha256Hex(payload)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	canonicalHeaders, signedHeaders := buildCanonicalHeaders(req)
	canonicalQuerystring := ""
	if req.URL.RawQuery != "" {
		canonicalQuerystring = req.URL.RawQuery
	}
	canonicalRequest := strings.Join([]string{
		req.Method,
		req.URL.Path,
		canonicalQuerystring,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	// Create string to sign.
	credentialScope := datestamp + "/" + region + "/" + service + "/aws4_request"
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzdate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	// Calculate signature.
	signingKey := deriveSigningKey(secretKey, datestamp, region, service)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	// Set authorization header.
	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKey, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)

	return nil
}

// parseServiceHost extracts service and region from an AWS hostname.
// For standard hosts like "ec2.us-east-1.amazonaws.com", returns ("ec2", "us-east-1").
// For S3 path-style "s3.us-east-1.amazonaws.com", returns ("s3", "us-east-1").
// For global endpoints like "monitoring.us-east-1.amazonaws.com", returns ("monitoring", "us-east-1").
func parseServiceHost(host string) (service, region string) {
	host = strings.TrimSuffix(host, ".amazonaws.com")
	parts := strings.SplitN(host, ".", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return host, "us-east-1"
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func deriveSigningKey(secret, datestamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(datestamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func buildCanonicalHeaders(req *http.Request) (canonicalHeaders, signedHeaders string) {
	headers := make(map[string]string)
	var names []string
	for name := range req.Header {
		lower := strings.ToLower(name)
		if lower == "host" || lower == "content-type" || strings.HasPrefix(lower, "x-amz-") {
			headers[lower] = strings.TrimSpace(req.Header.Get(name))
			names = append(names, lower)
		}
	}
	sort.Strings(names)

	var canonical, signed []string
	for _, name := range names {
		canonical = append(canonical, name+":"+headers[name])
		signed = append(signed, name)
	}

	return strings.Join(canonical, "\n") + "\n", strings.Join(signed, ";")
}
