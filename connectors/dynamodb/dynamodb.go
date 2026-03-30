// Package dynamodb implements the Amazon DynamoDB connector for the Permission Slip
// connector execution layer. It uses AWS SDK for Go v2. Credentials are static
// access keys (and optional session token) stored in the vault, matching the
// existing aws connector pattern.
//
// Security model:
//   - Table allowlists constrain which tables an agent may access
//   - Optional read/write attribute allowlists limit projection and put payloads
//   - Query limits cap items scanned per request
package dynamodb

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	defaultTimeout    = 30 * time.Second
	defaultQueryLimit = 100
	maxQueryLimit     = 1000
	maxAllowedTables  = 64
	maxAttrAllowlist  = 100
)

// dynamoAPI is the subset of DynamoDB client methods used by this connector (test doubles).
type dynamoAPI interface {
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

// DynamoDBConnector owns shared configuration for DynamoDB actions.
type DynamoDBConnector struct {
	timeout   time.Duration
	newClient func(ctx context.Context, region string, creds connectors.Credentials) (dynamoAPI, error)
}

// New creates a DynamoDBConnector with production defaults.
func New() *DynamoDBConnector {
	c := &DynamoDBConnector{timeout: defaultTimeout}
	c.newClient = c.buildClient
	return c
}

func newForTest(api dynamoAPI) *DynamoDBConnector {
	return &DynamoDBConnector{
		timeout: defaultTimeout,
		newClient: func(context.Context, string, connectors.Credentials) (dynamoAPI, error) {
			return api, nil
		},
	}
}

// ID returns "dynamodb", matching connectors.id in the database.
func (c *DynamoDBConnector) ID() string { return "dynamodb" }

//go:embed logo.svg
var logoSVG string

// Manifest returns connector metadata for DB seeding.
func (c *DynamoDBConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "dynamodb",
		Name:        "Amazon DynamoDB",
		Description: "DynamoDB integration for key/value and query access in AWS serverless stacks",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:      "dynamodb.get_item",
				Name:            "Get Item",
				Description:     "Read a single item by primary key with optional projection",
				RiskLevel:       "low",
				DisplayTemplate: "Get item from {{table}} ({{region}})",
				Preview: &connectors.ActionPreview{
					Layout: "record",
					Fields: map[string]string{"title": "table", "subtitle": "region"},
				},
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["region", "table", "key", "allowed_tables"],
					"properties": {
						"region": {
							"type": "string",
							"description": "AWS region (e.g. us-east-1)"
						},
						"table": {
							"type": "string",
							"description": "DynamoDB table name"
						},
						"key": {
							"type": "object",
							"description": "Primary key attributes as JSON (string, number, or boolean values)"
						},
						"consistent_read": {
							"type": "boolean",
							"description": "Use strongly consistent read (default false)"
						},
						"projection": {
							"type": "object",
							"description": "Attribute projection map (attribute name → true to include)"
						},
						"allowed_tables": {
							"type": "array",
							"items": {"type": "string"},
							"minItems": 1,
							"maxItems": 64,
							"description": "Table allowlist — request is rejected if table is not listed"
						},
						"allowed_read_attributes": {
							"type": "array",
							"items": {"type": "string"},
							"maxItems": 100,
							"description": "When set, only these attributes may be read; projection must be a subset (or omitted to return all allowed)"
						}
					}
				}`)),
			},
			{
				ActionType:      "dynamodb.put_item",
				Name:            "Put Item",
				Description:     "Create or replace an item with JSON attributes",
				RiskLevel:       "medium",
				DisplayTemplate: "Put item into {{table}} ({{region}})",
				Preview: &connectors.ActionPreview{
					Layout: "record",
					Fields: map[string]string{"title": "table", "subtitle": "region"},
				},
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["region", "table", "item", "allowed_tables"],
					"properties": {
						"region": {"type": "string", "description": "AWS region (e.g. us-east-1)"},
						"table": {"type": "string", "description": "DynamoDB table name"},
						"item": {
							"type": "object",
							"description": "Item attributes including primary key attributes"
						},
						"condition_expression": {
							"type": "string",
							"description": "Optional DynamoDB condition expression"
						},
						"expression_attribute_names": {
							"type": "object",
							"description": "Placeholder map for expression attribute names"
						},
						"expression_attribute_values": {
							"type": "object",
							"description": "Placeholder map for expression attribute values (JSON types)"
						},
						"allowed_tables": {
							"type": "array",
							"items": {"type": "string"},
							"minItems": 1,
							"maxItems": 64,
							"description": "Table allowlist — request is rejected if table is not listed"
						},
						"allowed_write_attributes": {
							"type": "array",
							"items": {"type": "string"},
							"maxItems": 100,
							"description": "When set, item may only include these attribute names (plus keys required by the table)"
						}
					}
				}`)),
			},
			{
				ActionType:      "dynamodb.delete_item",
				Name:            "Delete Item",
				Description:     "Delete an item by primary key",
				RiskLevel:       "high",
				DisplayTemplate: "Delete item from {{table}} ({{region}})",
				Preview: &connectors.ActionPreview{
					Layout: "record",
					Fields: map[string]string{"title": "table", "subtitle": "region"},
				},
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["region", "table", "key", "allowed_tables"],
					"properties": {
						"region": {"type": "string", "description": "AWS region (e.g. us-east-1)"},
						"table": {"type": "string", "description": "DynamoDB table name"},
						"key": {
							"type": "object",
							"description": "Primary key attributes"
						},
						"condition_expression": {"type": "string", "description": "Optional condition expression"},
						"expression_attribute_names": {"type": "object"},
						"expression_attribute_values": {"type": "object"},
						"allowed_tables": {
							"type": "array",
							"items": {"type": "string"},
							"minItems": 1,
							"maxItems": 64,
							"description": "Table allowlist — request is rejected if table is not listed"
						},
						"allowed_write_attributes": {
							"type": "array",
							"items": {"type": "string"},
							"maxItems": 100,
							"description": "When set, condition expression attribute names must reference only these attributes"
						},
						"allowed_read_attributes": {
							"type": "array",
							"items": {"type": "string"},
							"maxItems": 100,
							"description": "When set, previous_item only returns these attributes"
						}
					}
				}`)),
			},
			{
				ActionType:      "dynamodb.query",
				Name:            "Query",
				Description:     "Query items by partition key with optional sort key conditions",
				RiskLevel:       "low",
				DisplayTemplate: "Query {{table}} by {{partition_key_name}} ({{region}})",
				Preview: &connectors.ActionPreview{
					Layout: "record",
					Fields: map[string]string{"title": "table", "subtitle": "partition_key_name"},
				},
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["region", "table", "partition_key_name", "partition_key_value", "allowed_tables"],
					"properties": {
						"region": {"type": "string", "description": "AWS region (e.g. us-east-1)"},
						"table": {"type": "string", "description": "DynamoDB table name"},
						"index_name": {
							"type": "string",
							"description": "Optional GSI or LSI name"
						},
						"partition_key_name": {"type": "string", "description": "Partition key attribute name"},
						"partition_key_value": {"description": "Partition key value (string, number, or boolean)"},
						"sort_key_name": {"type": "string", "description": "Sort key attribute name when using sort conditions"},
						"sort_key_value": {"description": "Sort key value for equality condition"},
						"sort_key_condition": {
							"type": "string",
							"enum": ["eq", "lt", "lte", "gt", "gte", "begins_with", "between"],
							"description": "Sort key condition operator (use with sort_key_name)"
						},
						"sort_key_between": {
							"type": "array",
							"minItems": 2,
							"maxItems": 2,
							"description": "Two values for between condition [lower, upper]",
							"items": {}
						},
						"projection": {"type": "object", "description": "Attribute projection map"},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 1000,
							"default": 100,
							"description": "Maximum items to return per request"
						},
						"scan_index_forward": {
							"type": "boolean",
							"description": "Ascending order when true (default true for queries)"
						},
						"exclusive_start_key": {
							"type": "object",
							"description": "Pagination token from a previous query response"
						},
						"allowed_tables": {
							"type": "array",
							"items": {"type": "string"},
							"minItems": 1,
							"maxItems": 64,
							"description": "Table allowlist — request is rejected if table is not listed"
						},
						"allowed_read_attributes": {
							"type": "array",
							"items": {"type": "string"},
							"maxItems": 100,
							"description": "When set, only these attributes may be read"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "dynamodb",
				AuthType:        "custom",
				InstructionsURL: "https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_dynamodb_read_one_table",
				ActionType:  "dynamodb.get_item",
				Name:        "Read from one table",
				Description: "Agent may get items from a specific table only.",
				Parameters:  json.RawMessage(`{"region":"us-east-1","table":"MyTable","key":"*","allowed_tables":["MyTable"],"projection":"*"}`),
			},
			{
				ID:          "tpl_dynamodb_put_one_table",
				ActionType:  "dynamodb.put_item",
				Name:        "Write to one table",
				Description: "Agent may put items into a specific table.",
				Parameters:  json.RawMessage(`{"region":"us-east-1","table":"MyTable","item":"*","allowed_tables":["MyTable"]}`),
			},
			{
				ID:          "tpl_dynamodb_query_pk",
				ActionType:  "dynamodb.query",
				Name:        "Query by partition key",
				Description: "Agent may query a table by partition key with optional limit.",
				Parameters:  json.RawMessage(`{"region":"us-east-1","table":"MyTable","partition_key_name":"pk","partition_key_value":"*","allowed_tables":["MyTable"],"limit":"*"}`),
			},
		},
	}
}

// Actions returns registered handlers.
func (c *DynamoDBConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"dynamodb.get_item":    &getItemAction{conn: c},
		"dynamodb.put_item":    &putItemAction{conn: c},
		"dynamodb.delete_item": &deleteItemAction{conn: c},
		"dynamodb.query":       &queryAction{conn: c},
	}
}

// ValidateCredentials ensures static AWS keys are present (same as aws connector).
// Optional: endpoint_url (e.g. http://localhost:4566 for LocalStack), session_token (STS).
func (c *DynamoDBConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	accessKey, ok := creds.Get("access_key_id")
	if !ok || accessKey == "" {
		return &connectors.ValidationError{Message: "missing required credential: access_key_id"}
	}
	secretKey, ok := creds.Get("secret_access_key")
	if !ok || secretKey == "" {
		return &connectors.ValidationError{Message: "missing required credential: secret_access_key"}
	}
	if ep, ok := creds.Get("endpoint_url"); ok && strings.TrimSpace(ep) != "" {
		if _, err := parseDynamoEndpoint(ep); err != nil {
			return err
		}
	}
	return nil
}

func parseDynamoEndpoint(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", &connectors.ValidationError{Message: "endpoint_url must be a full URL with host (e.g. https://dynamodb.us-east-1.amazonaws.com or http://localhost:4566)"}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", &connectors.ValidationError{Message: "endpoint_url scheme must be http or https"}
	}
	return strings.TrimRight(raw, "/"), nil
}

func (c *DynamoDBConnector) buildClient(ctx context.Context, region string, creds connectors.Credentials) (dynamoAPI, error) {
	accessKey, _ := creds.Get("access_key_id")
	secretKey, _ := creds.Get("secret_access_key")
	sessionToken, _ := creds.Get("session_token")

	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, sessionToken)),
	)
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("loading AWS config: %v", err)}
	}

	var baseEndpoint *string
	if ep, ok := creds.Get("endpoint_url"); ok && strings.TrimSpace(ep) != "" {
		trimmed, perr := parseDynamoEndpoint(ep)
		if perr != nil {
			return nil, perr
		}
		baseEndpoint = aws.String(trimmed)
	}

	return dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		if c.timeout > 0 {
			o.HTTPClient = &http.Client{Timeout: c.timeout}
		}
		if baseEndpoint != nil {
			o.BaseEndpoint = baseEndpoint
		}
	}), nil
}
