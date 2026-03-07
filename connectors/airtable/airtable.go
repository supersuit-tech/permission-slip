// Package airtable implements the Airtable connector for the Permission Slip
// connector execution layer. It uses the Airtable REST API with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
package airtable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultBaseURL = "https://api.airtable.com/v0"
	metaBaseURL    = "https://api.airtable.com/v0/meta"
	defaultTimeout = 30 * time.Second
	credKeyToken   = "api_token"
	tokenPrefix    = "pat"

	// defaultRetryAfter is used when the Airtable API returns a rate limit
	// response without a Retry-After header.
	defaultRetryAfter = 30 * time.Second

	// maxResponseBytes caps the response body at 10 MB.
	maxResponseBytes = 10 << 20 // 10 MB
)

// AirtableConnector owns the shared HTTP client and base URL used by all
// Airtable actions.
type AirtableConnector struct {
	client  *http.Client
	baseURL string
	metaURL string
}

// New creates an AirtableConnector with sensible defaults.
func New() *AirtableConnector {
	return &AirtableConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
		metaURL: metaBaseURL,
	}
}

// newForTest creates an AirtableConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *AirtableConnector {
	return &AirtableConnector{
		client:  client,
		baseURL: baseURL,
		metaURL: baseURL + "/meta",
	}
}

// ID returns "airtable", matching the connectors.id in the database.
func (c *AirtableConnector) ID() string { return "airtable" }

// Manifest returns the connector's metadata manifest.
func (c *AirtableConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "airtable",
		Name:        "Airtable",
		Description: "Airtable integration for structured data and no-code databases",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "airtable.list_bases",
				Name:        "List Bases",
				Description: "List all bases accessible to the authenticated user",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"offset": {
							"type": "string",
							"description": "Pagination offset from a previous response"
						}
					}
				}`)),
			},
			{
				ActionType:  "airtable.list_records",
				Name:        "List Records",
				Description: "List records from an Airtable table with optional filtering, sorting, and pagination",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["base_id", "table"],
					"properties": {
						"base_id": {
							"type": "string",
							"description": "Airtable base ID (starts with 'app')"
						},
						"table": {
							"type": "string",
							"description": "Table name or ID"
						},
						"fields": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Only return these fields (column names)"
						},
						"filter_by_formula": {
							"type": "string",
							"description": "Airtable formula to filter records (e.g. \"{Status} = 'Active'\")"
						},
						"max_records": {
							"type": "integer",
							"description": "Maximum total records to return"
						},
						"page_size": {
							"type": "integer",
							"description": "Records per page (max 100, default 100)"
						},
						"sort": {
							"type": "array",
							"items": {
								"type": "object",
								"required": ["field"],
								"properties": {
									"field": {"type": "string", "description": "Field name to sort by"},
									"direction": {"type": "string", "enum": ["asc", "desc"], "description": "Sort direction (default: asc)"}
								}
							},
							"description": "Sort order for records"
						},
						"view": {
							"type": "string",
							"description": "Name or ID of a view to filter/sort by"
						},
						"offset": {
							"type": "string",
							"description": "Pagination offset from a previous response"
						}
					}
				}`)),
			},
			{
				ActionType:  "airtable.get_record",
				Name:        "Get Record",
				Description: "Get a single record by its ID",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["base_id", "table", "record_id"],
					"properties": {
						"base_id": {
							"type": "string",
							"description": "Airtable base ID (starts with 'app')"
						},
						"table": {
							"type": "string",
							"description": "Table name or ID"
						},
						"record_id": {
							"type": "string",
							"description": "Record ID (starts with 'rec')"
						}
					}
				}`)),
			},
			{
				ActionType:  "airtable.create_records",
				Name:        "Create Records",
				Description: "Create one or more records in an Airtable table (batch up to 10)",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["base_id", "table", "records"],
					"properties": {
						"base_id": {
							"type": "string",
							"description": "Airtable base ID (starts with 'app')"
						},
						"table": {
							"type": "string",
							"description": "Table name or ID"
						},
						"records": {
							"type": "array",
							"minItems": 1,
							"maxItems": 10,
							"items": {
								"type": "object",
								"required": ["fields"],
								"properties": {
									"fields": {
										"type": "object",
										"description": "Field name-value pairs for the record"
									}
								}
							},
							"description": "Records to create (1-10)"
						}
					}
				}`)),
			},
			{
				ActionType:  "airtable.update_records",
				Name:        "Update Records",
				Description: "Update one or more existing records with partial updates (batch up to 10)",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["base_id", "table", "records"],
					"properties": {
						"base_id": {
							"type": "string",
							"description": "Airtable base ID (starts with 'app')"
						},
						"table": {
							"type": "string",
							"description": "Table name or ID"
						},
						"records": {
							"type": "array",
							"minItems": 1,
							"maxItems": 10,
							"items": {
								"type": "object",
								"required": ["id", "fields"],
								"properties": {
									"id": {
										"type": "string",
										"description": "Record ID to update (starts with 'rec')"
									},
									"fields": {
										"type": "object",
										"description": "Field name-value pairs to update"
									}
								}
							},
							"description": "Records to update (1-10)"
						}
					}
				}`)),
			},
			{
				ActionType:  "airtable.delete_records",
				Name:        "Delete Records",
				Description: "Delete one or more records from an Airtable table (batch up to 10)",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["base_id", "table", "record_ids"],
					"properties": {
						"base_id": {
							"type": "string",
							"description": "Airtable base ID (starts with 'app')"
						},
						"table": {
							"type": "string",
							"description": "Table name or ID"
						},
						"record_ids": {
							"type": "array",
							"minItems": 1,
							"maxItems": 10,
							"items": {"type": "string"},
							"description": "Record IDs to delete (each starts with 'rec')"
						}
					}
				}`)),
			},
			{
				ActionType:  "airtable.search_records",
				Name:        "Search Records",
				Description: "Search records using an Airtable formula filter",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["base_id", "table", "formula"],
					"properties": {
						"base_id": {
							"type": "string",
							"description": "Airtable base ID (starts with 'app')"
						},
						"table": {
							"type": "string",
							"description": "Table name or ID"
						},
						"formula": {
							"type": "string",
							"description": "Airtable formula to filter records (e.g. \"SEARCH('John', {Name})\" or \"{Status} = 'Active'\")"
						},
						"fields": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Only return these fields (column names)"
						},
						"max_records": {
							"type": "integer",
							"description": "Maximum total records to return (default: 100)"
						},
						"sort": {
							"type": "array",
							"items": {
								"type": "object",
								"required": ["field"],
								"properties": {
									"field": {"type": "string", "description": "Field name to sort by"},
									"direction": {"type": "string", "enum": ["asc", "desc"], "description": "Sort direction (default: asc)"}
								}
							},
							"description": "Sort order for results"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "airtable", AuthType: "api_key", InstructionsURL: "https://airtable.com/create/tokens"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_airtable_list_bases",
				ActionType:  "airtable.list_bases",
				Name:        "List all bases",
				Description: "Agent can list all accessible Airtable bases.",
				Parameters:  json.RawMessage(`{"offset":"*"}`),
			},
			{
				ID:          "tpl_airtable_read_records",
				ActionType:  "airtable.list_records",
				Name:        "Read records from any table",
				Description: "Agent can read records from any table in any base.",
				Parameters:  json.RawMessage(`{"base_id":"*","table":"*","fields":"*","filter_by_formula":"*","max_records":"*","page_size":"*","sort":"*","view":"*","offset":"*"}`),
			},
			{
				ID:          "tpl_airtable_get_record",
				ActionType:  "airtable.get_record",
				Name:        "Get any record",
				Description: "Agent can get any record by ID from any table.",
				Parameters:  json.RawMessage(`{"base_id":"*","table":"*","record_id":"*"}`),
			},
			{
				ID:          "tpl_airtable_create_records",
				ActionType:  "airtable.create_records",
				Name:        "Create records",
				Description: "Agent can create records in any table.",
				Parameters:  json.RawMessage(`{"base_id":"*","table":"*","records":"*"}`),
			},
			{
				ID:          "tpl_airtable_update_records",
				ActionType:  "airtable.update_records",
				Name:        "Update records",
				Description: "Agent can update records in any table.",
				Parameters:  json.RawMessage(`{"base_id":"*","table":"*","records":"*"}`),
			},
			{
				ID:          "tpl_airtable_delete_records",
				ActionType:  "airtable.delete_records",
				Name:        "Delete records",
				Description: "Agent can delete records from any table.",
				Parameters:  json.RawMessage(`{"base_id":"*","table":"*","record_ids":"*"}`),
			},
			{
				ID:          "tpl_airtable_search_records",
				ActionType:  "airtable.search_records",
				Name:        "Search records",
				Description: "Agent can search records in any table using formulas.",
				Parameters:  json.RawMessage(`{"base_id":"*","table":"*","formula":"*","fields":"*","max_records":"*","sort":"*"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *AirtableConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"airtable.list_bases":      &listBasesAction{conn: c},
		"airtable.list_records":    &listRecordsAction{conn: c},
		"airtable.get_record":      &getRecordAction{conn: c},
		"airtable.create_records":  &createRecordsAction{conn: c},
		"airtable.update_records":  &updateRecordsAction{conn: c},
		"airtable.delete_records":  &deleteRecordsAction{conn: c},
		"airtable.search_records":  &searchRecordsAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty personal access token with the required pat prefix.
func (c *AirtableConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_token"}
	}
	if len(token) < len(tokenPrefix) || token[:len(tokenPrefix)] != tokenPrefix {
		return &connectors.ValidationError{Message: "api_token must be a personal access token starting with \"pat\""}
	}
	return nil
}

// validateBaseID checks that a base_id looks like a valid Airtable base ID.
func validateBaseID(baseID string) error {
	if baseID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: base_id"}
	}
	if len(baseID) < 4 || baseID[:3] != "app" {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid base_id %q: expected an Airtable base ID starting with 'app'", baseID),
		}
	}
	return nil
}

// validateRecordID checks that a record_id looks like a valid Airtable record ID.
func validateRecordID(recordID string) error {
	if recordID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: record_id"}
	}
	if len(recordID) < 4 || recordID[:3] != "rec" {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid record_id %q: expected an Airtable record ID starting with 'rec'", recordID),
		}
	}
	return nil
}

// doRequest executes an HTTP request against the Airtable API with auth headers,
// handles rate limiting and timeouts, and unmarshals the response into dest.
func (c *AirtableConnector) doRequest(ctx context.Context, method, url string, creds connectors.Credentials, body io.Reader, dest any) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "api_token credential is missing or empty"}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Airtable API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Airtable API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Airtable API request failed: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(resp.Header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    "Airtable API rate limit exceeded (5 requests/second per base)",
			RetryAfter: retryAfter,
		}
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return &connectors.AuthError{Message: fmt.Sprintf("Airtable auth error (HTTP %d): %s", resp.StatusCode, string(respBody))}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mapAirtableError(resp.StatusCode, respBody)
	}

	if dest != nil {
		if err := json.Unmarshal(respBody, dest); err != nil {
			return &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    "failed to decode Airtable API response",
			}
		}
	}

	return nil
}

// airtableErrorResponse is the error envelope from the Airtable API.
type airtableErrorResponse struct {
	Error *airtableErrorDetail `json:"error,omitempty"`
}

type airtableErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// mapAirtableError converts an Airtable API error response to the appropriate
// connector error type.
func mapAirtableError(statusCode int, body []byte) error {
	var errResp airtableErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil || errResp.Error == nil {
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Airtable API error (HTTP %d): %s", statusCode, string(body)),
		}
	}

	msg := fmt.Sprintf("Airtable API error: %s - %s", errResp.Error.Type, errResp.Error.Message)

	switch errResp.Error.Type {
	case "AUTHENTICATION_REQUIRED", "INVALID_PERMISSIONS_OR_MODEL_NOT_FOUND":
		return &connectors.AuthError{Message: msg}
	case "NOT_FOUND":
		return &connectors.ExternalError{StatusCode: 404, Message: msg}
	case "INVALID_REQUEST_UNKNOWN", "INVALID_VALUE_FOR_COLUMN", "CANNOT_UPDATE_COMPUTED_FIELD":
		return &connectors.ValidationError{Message: msg}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: msg}
	}
}
