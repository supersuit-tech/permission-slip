// Package monday implements the Monday.com connector for the Permission Slip
// connector execution layer. It uses the Monday.com GraphQL API with plain
// net/http (no third-party SDK) to keep the dependency footprint minimal.
package monday

import (
	"bytes"
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
	defaultBaseURL = "https://api.monday.com/v2"
	defaultTimeout = 30 * time.Second
	credKeyToken   = "api_key"

	// defaultRetryAfter is used when the Monday.com API returns a rate limit
	// response without a Retry-After header (or an unparseable one).
	defaultRetryAfter = 60 * time.Second

	// maxResponseBytes caps the Monday.com API response body at 10 MB.
	maxResponseBytes = 10 << 20 // 10 MB
)

// MondayConnector owns the shared HTTP client and base URL used by all
// Monday.com actions.
type MondayConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a MondayConnector with sensible defaults (30s timeout,
// https://api.monday.com/v2 base URL).
func New() *MondayConnector {
	return &MondayConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a MondayConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *MondayConnector {
	return &MondayConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "monday", matching the connectors.id in the database.
func (c *MondayConnector) ID() string { return "monday" }

// Manifest returns the connector's metadata manifest.
func (c *MondayConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "monday",
		Name:        "Monday.com",
		Description: "Monday.com integration for project management",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "monday.create_item",
				Name:        "Create Item",
				Description: "Create a new item on a Monday.com board",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["board_id", "item_name"],
					"properties": {
						"board_id": {
							"type": "string",
							"description": "The board ID to create the item on"
						},
						"item_name": {
							"type": "string",
							"description": "Name of the new item"
						},
						"column_values": {
							"type": "object",
							"description": "JSON object mapping column IDs to values, e.g. {\"status\": {\"label\": \"Working on it\"}, \"date\": {\"date\": \"2024-01-15\"}}"
						},
						"group_id": {
							"type": "string",
							"description": "Group ID to create the item in (use the group's unique ID, not its display name)"
						}
					}
				}`)),
			},
			{
				ActionType:  "monday.update_item",
				Name:        "Update Item",
				Description: "Update column values on an existing Monday.com item",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["board_id", "item_id", "column_values"],
					"properties": {
						"board_id": {
							"type": "string",
							"description": "The board ID containing the item"
						},
						"item_id": {
							"type": "string",
							"description": "The item ID to update"
						},
						"column_values": {
							"type": "object",
							"description": "JSON object mapping column IDs to new values"
						}
					}
				}`)),
			},
			{
				ActionType:  "monday.add_update",
				Name:        "Add Update",
				Description: "Add an update (comment) to a Monday.com item",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["item_id", "body"],
					"properties": {
						"item_id": {
							"type": "string",
							"description": "The item ID to add the update to"
						},
						"body": {
							"type": "string",
							"description": "Update text content (supports HTML)"
						}
					}
				}`)),
			},
			{
				ActionType:  "monday.create_subitem",
				Name:        "Create Subitem",
				Description: "Create a subitem under an existing Monday.com item",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["parent_item_id", "item_name"],
					"properties": {
						"parent_item_id": {
							"type": "string",
							"description": "The parent item ID to create the subitem under"
						},
						"item_name": {
							"type": "string",
							"description": "Name of the new subitem"
						},
						"column_values": {
							"type": "object",
							"description": "JSON object mapping column IDs to values"
						}
					}
				}`)),
			},
			{
				ActionType:  "monday.move_item_to_group",
				Name:        "Move Item to Group",
				Description: "Move an item to a different group on its board",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["item_id", "group_id"],
					"properties": {
						"item_id": {
							"type": "string",
							"description": "The item ID to move"
						},
						"group_id": {
							"type": "string",
							"description": "The target group ID (e.g. 'done', 'in_progress')"
						}
					}
				}`)),
			},
			{
				ActionType:  "monday.search_items",
				Name:        "Search Items",
				Description: "Search and filter items on a Monday.com board",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["board_id"],
					"properties": {
						"board_id": {
							"type": "string",
							"description": "The board ID to search on"
						},
						"query": {
							"type": "string",
							"description": "Text search query"
						},
						"column_id": {
							"type": "string",
							"description": "Column ID to filter by (use with column_value)"
						},
						"column_value": {
							"type": "string",
							"description": "Column value to filter by (use with column_id)"
						},
						"limit": {
							"type": "integer",
							"default": 20,
							"description": "Maximum number of items to return (default 20)"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "monday",
				AuthType:        "api_key",
				InstructionsURL: "https://developer.monday.com/apps/docs/manage-access-tokens",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_monday_create_item_on_board",
				ActionType:  "monday.create_item",
				Name:        "Create items on a specific board",
				Description: "Replace the board_id with your board's numeric ID. Agent can create items with any name and column values.",
				Parameters:  json.RawMessage(`{"board_id":"1234567890","item_name":"*","column_values":"*","group_id":"*"}`),
			},
			{
				ID:          "tpl_monday_create_item_any",
				ActionType:  "monday.create_item",
				Name:        "Create items on any board",
				Description: "Agent can create items on any board with any values.",
				Parameters:  json.RawMessage(`{"board_id":"*","item_name":"*","column_values":"*","group_id":"*"}`),
			},
			{
				ID:          "tpl_monday_update_item",
				ActionType:  "monday.update_item",
				Name:        "Update items on a specific board",
				Description: "Replace the board_id with your board's numeric ID. Agent can update column values on any item in that board.",
				Parameters:  json.RawMessage(`{"board_id":"1234567890","item_id":"*","column_values":"*"}`),
			},
			{
				ID:          "tpl_monday_add_update",
				ActionType:  "monday.add_update",
				Name:        "Add updates to items",
				Description: "Agent can add comments and updates to any item.",
				Parameters:  json.RawMessage(`{"item_id":"*","body":"*"}`),
			},
			{
				ID:          "tpl_monday_create_subitem",
				ActionType:  "monday.create_subitem",
				Name:        "Create subitems",
				Description: "Agent can create subitems under any item.",
				Parameters:  json.RawMessage(`{"parent_item_id":"*","item_name":"*","column_values":"*"}`),
			},
			{
				ID:          "tpl_monday_move_to_group",
				ActionType:  "monday.move_item_to_group",
				Name:        "Move items between groups",
				Description: "Agent can move items to any group (e.g. status changes like moving to 'Done').",
				Parameters:  json.RawMessage(`{"item_id":"*","group_id":"*"}`),
			},
			{
				ID:          "tpl_monday_search_items",
				ActionType:  "monday.search_items",
				Name:        "Search items on any board",
				Description: "Agent can search and filter items. Use query for text search or column_id+column_value for filtering.",
				Parameters:  json.RawMessage(`{"board_id":"*","query":"*","column_id":"*","column_value":"*","limit":20}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *MondayConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"monday.create_item":       &createItemAction{conn: c},
		"monday.update_item":       &updateItemAction{conn: c},
		"monday.add_update":        &addUpdateAction{conn: c},
		"monday.create_subitem":    &createSubitemAction{conn: c},
		"monday.move_item_to_group": &moveItemToGroupAction{conn: c},
		"monday.search_items":      &searchItemsAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty api_key.
func (c *MondayConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_key"}
	}
	return nil
}

// graphQLRequest is the Monday.com GraphQL request body.
type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// graphQLResponse is the common envelope for Monday.com GraphQL responses.
type graphQLResponse struct {
	Data   json.RawMessage `json:"data,omitempty"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// query sends a GraphQL request to the Monday.com API and unmarshals the
// response data into dest.
func (c *MondayConnector) query(ctx context.Context, creds connectors.Credentials, gqlQuery string, variables map[string]any, dest any) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "api_key credential is missing or empty"}
	}

	reqBody := graphQLRequest{
		Query:     gqlQuery,
		Variables: variables,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("marshaling request body: %v", err)}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("creating request: %v", err)}
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Monday.com API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Monday.com API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Monday.com API request failed: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(resp.Header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    "Monday.com API rate limit exceeded",
			RetryAfter: retryAfter,
		}
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return &connectors.AuthError{Message: "Monday.com auth error: invalid or expired API key"}
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	// Handle HTTP 400 as a validation error (e.g. malformed query).
	if resp.StatusCode == http.StatusBadRequest {
		msg := extractErrorMessage(respBody)
		if msg == "" {
			msg = "Monday.com API rejected the request"
		}
		return &connectors.ValidationError{Message: fmt.Sprintf("Monday.com validation error: %s", msg)}
	}

	// Catch other non-2xx status codes not already handled above.
	if resp.StatusCode >= 400 {
		msg := extractErrorMessage(respBody)
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("Monday.com API error: %s", msg),
		}
	}

	var gqlResp graphQLResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    "failed to decode Monday.com API response",
		}
	}

	// Check for top-level error fields (e.g. auth errors returned as 200).
	if gqlResp.ErrorCode != "" {
		return mapMondayError(gqlResp.ErrorCode, gqlResp.ErrorMessage)
	}

	// Check for GraphQL errors array.
	if len(gqlResp.Errors) > 0 {
		msg := gqlResp.Errors[0].Message
		return &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("Monday.com API error: %s", msg),
		}
	}

	if dest != nil && gqlResp.Data != nil {
		if err := json.Unmarshal(gqlResp.Data, dest); err != nil {
			return &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    fmt.Sprintf("failed to decode Monday.com response data: %v", err),
			}
		}
	}

	return nil
}

// isValidMondayID checks that an ID is a non-empty numeric string.
// Monday.com IDs are always numeric, so rejecting non-numeric values
// prevents unexpected API behavior.
func isValidMondayID(id string) bool {
	if id == "" {
		return false
	}
	for _, c := range id {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// stringifyColumnValues marshals a column_values map to a JSON string,
// which is the format Monday.com's GraphQL API expects.
func stringifyColumnValues(cv map[string]any) (string, error) {
	data, err := json.Marshal(cv)
	if err != nil {
		return "", &connectors.ValidationError{Message: fmt.Sprintf("invalid column_values: %v", err)}
	}
	return string(data), nil
}

// extractErrorMessage tries to pull an error message from a Monday.com
// error response body. Returns empty string if the body can't be parsed.
func extractErrorMessage(body []byte) string {
	var envelope struct {
		ErrorMessage string `json:"error_message"`
		Errors       []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if json.Unmarshal(body, &envelope) != nil {
		return ""
	}
	if envelope.ErrorMessage != "" {
		return envelope.ErrorMessage
	}
	if len(envelope.Errors) > 0 && envelope.Errors[0].Message != "" {
		return envelope.Errors[0].Message
	}
	return ""
}

// mapMondayError converts a Monday.com error code to the appropriate
// connector error type.
func mapMondayError(code, message string) error {
	if message == "" {
		message = code
	}
	switch code {
	case "UserUnauthorizedException", "NotAuthenticated", "invalid_token":
		return &connectors.AuthError{Message: fmt.Sprintf("Monday.com auth error: %s", message)}
	case "RateLimitExceeded", "ComplexityBudgetExhausted":
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Monday.com rate limit: %s", message),
			RetryAfter: defaultRetryAfter,
		}
	default:
		return &connectors.ExternalError{
			StatusCode: 200,
			Message:    fmt.Sprintf("Monday.com API error: %s", message),
		}
	}
}
