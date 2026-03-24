// Package supabase implements the Supabase connector for the Permission Slip
// connector execution layer. It uses the Supabase PostgREST API with plain
// net/http — no SQL driver or third-party SDK needed.
//
// Security model:
//   - Table allowlists constrain which tables an agent can access
//   - API key scoping: anon key respects RLS, service_role key bypasses it
//   - All filter values are passed as PostgREST query parameters (never interpolated)
//   - Response size is capped to prevent runaway reads
package supabase

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultTimeout    = 30 * time.Second
	maxResponseBytes  = 10 << 20 // 10 MB
	defaultMaxRows    = 1000
	defaultRetryAfter = 30 * time.Second

	credKeyURL    = "supabase_url"
	credKeyAPIKey = "api_key"
)

// SupabaseConnector owns the shared HTTP client used by all Supabase actions.
type SupabaseConnector struct {
	client *http.Client
}

// New creates a SupabaseConnector with sensible defaults.
func New() *SupabaseConnector {
	return &SupabaseConnector{
		client: &http.Client{Timeout: defaultTimeout},
	}
}

// newForTest creates a SupabaseConnector pointing at a test server.
func newForTest(client *http.Client) *SupabaseConnector {
	return &SupabaseConnector{client: client}
}

// ID returns "supabase", matching the connectors.id in the database.
func (c *SupabaseConnector) ID() string { return "supabase" }

//go:embed logo.svg
var logoSVG string

// Manifest returns the connector's metadata manifest.
func (c *SupabaseConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "supabase",
		Name:        "Supabase",
		Description: "Read and write Supabase tables via PostgREST with RLS support, table allowlists, and API key scoping",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "supabase.read",
				Name:        "Read Rows",
				Description: "Read rows from a Supabase table with optional filters, column selection, ordering, and pagination via PostgREST.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["table"],
					"properties": {
						"table": {
							"type": "string",
							"description": "Table name to read from"
						},
						"select": {
							"type": "string",
							"description": "Columns to return (PostgREST select syntax, e.g. 'id,name' or '*'). Default: *"
						},
						"filters": {
							"type": "object",
							"description": "PostgREST filter conditions as column-operator pairs. Keys are column names, values are 'operator.value' strings (e.g. {\"age\": \"gte.18\", \"status\": \"eq.active\"})"
						},
						"order": {
							"type": "string",
							"description": "Order results (e.g. 'created_at.desc' or 'name.asc,id.desc')"
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 10000,
							"description": "Maximum number of rows to return (default: 1000)"
						},
						"offset": {
							"type": "integer",
							"minimum": 0,
							"description": "Number of rows to skip for pagination"
						},
						"count_total": {
							"type": "boolean",
							"description": "If true, returns total_count (the total number of matching rows before limit/offset). Useful for building paginated UIs."
						},
						"allowed_tables": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Restrict access to these tables only. If set, requests for other tables are rejected."
						}
					}
				}`)),
			},
			{
				ActionType:  "supabase.insert",
				Name:        "Insert Rows",
				Description: "Insert one or more rows into a Supabase table via PostgREST. Supports returning inserted rows.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["table", "rows"],
					"properties": {
						"table": {
							"type": "string",
							"description": "Table name to insert into"
						},
						"rows": {
							"type": "array",
							"items": {"type": "object"},
							"minItems": 1,
							"maxItems": 1000,
							"description": "Array of row objects to insert (keys are column names)"
						},
						"returning": {
							"type": "string",
							"description": "Columns to return from inserted rows (PostgREST select syntax). Default: '*'"
						},
						"on_conflict": {
							"type": "string",
							"description": "Column(s) for upsert conflict resolution (e.g. 'id' or 'email')"
						},
						"allowed_tables": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Restrict access to these tables only"
						}
					}
				}`)),
			},
			{
				ActionType:  "supabase.update",
				Name:        "Update Rows",
				Description: "Update rows matching filters in a Supabase table via PostgREST. At least one filter is required.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["table", "set", "filters"],
					"properties": {
						"table": {
							"type": "string",
							"description": "Table name to update"
						},
						"set": {
							"type": "object",
							"description": "Column-value pairs to set on matching rows"
						},
						"filters": {
							"type": "object",
							"description": "PostgREST filter conditions (same format as read). At least one filter is required to prevent accidental full-table updates."
						},
						"returning": {
							"type": "string",
							"description": "Columns to return from updated rows (PostgREST select syntax). Default: '*'"
						},
						"allowed_tables": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Restrict access to these tables only"
						}
					}
				}`)),
			},
			{
				ActionType:  "supabase.delete",
				Name:        "Delete Rows",
				Description: "Delete rows matching filters from a Supabase table via PostgREST. At least one filter is required.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["table", "filters"],
					"properties": {
						"table": {
							"type": "string",
							"description": "Table name to delete from"
						},
						"filters": {
							"type": "object",
							"description": "PostgREST filter conditions (same format as read). At least one filter is required to prevent accidental full-table deletes."
						},
						"returning": {
							"type": "string",
							"description": "Columns to return from deleted rows (PostgREST select syntax). Default: '*'"
						},
						"allowed_tables": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Restrict access to these tables only"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "supabase",
				AuthType:        "custom",
				InstructionsURL: "https://supabase.com/docs/guides/api#api-url-and-keys",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_supabase_read",
				ActionType:  "supabase.read",
				Name:        "Read rows from any table",
				Description: "Agent can read rows from any allowed table with filters.",
				Parameters:  json.RawMessage(`{"table":"*","select":"*","filters":"*"}`),
			},
			{
				ID:          "tpl_supabase_insert",
				ActionType:  "supabase.insert",
				Name:        "Insert rows",
				Description: "Agent can insert rows into any allowed table.",
				Parameters:  json.RawMessage(`{"table":"*","rows":"*"}`),
			},
			{
				ID:          "tpl_supabase_update",
				ActionType:  "supabase.update",
				Name:        "Update rows",
				Description: "Agent can update rows in any allowed table with filter constraints.",
				Parameters:  json.RawMessage(`{"table":"*","set":"*","filters":"*"}`),
			},
			{
				ID:          "tpl_supabase_delete",
				ActionType:  "supabase.delete",
				Name:        "Delete rows",
				Description: "Agent can delete rows from any allowed table with filter constraints.",
				Parameters:  json.RawMessage(`{"table":"*","filters":"*"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *SupabaseConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"supabase.read":   &readAction{conn: c},
		"supabase.insert": &insertAction{conn: c},
		"supabase.update": &updateAction{conn: c},
		"supabase.delete": &deleteAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// valid Supabase URL and API key.
func (c *SupabaseConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	supabaseURL, ok := creds.Get(credKeyURL)
	if !ok || supabaseURL == "" {
		return &connectors.ValidationError{Message: "missing required credential: supabase_url"}
	}
	u, err := url.Parse(supabaseURL)
	if err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid supabase_url: %v", err)}
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return &connectors.ValidationError{Message: "supabase_url must use https:// (or http:// for local development)"}
	}

	apiKey, ok := creds.Get(credKeyAPIKey)
	if !ok || apiKey == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_key"}
	}
	if hasControlChars(apiKey) {
		return &connectors.ValidationError{Message: "api_key contains invalid control characters"}
	}
	return nil
}

// resolveConfig extracts the Supabase URL and API key from credentials.
func resolveConfig(creds connectors.Credentials) (baseURL, apiKey string, err error) {
	supabaseURL, ok := creds.Get(credKeyURL)
	if !ok || supabaseURL == "" {
		return "", "", &connectors.ValidationError{Message: "missing required credential: supabase_url"}
	}
	// Strip trailing slash for consistent URL building.
	baseURL = strings.TrimRight(supabaseURL, "/")

	apiKey, ok = creds.Get(credKeyAPIKey)
	if !ok || apiKey == "" {
		return "", "", &connectors.ValidationError{Message: "missing required credential: api_key"}
	}
	return baseURL, apiKey, nil
}

// restURL builds the PostgREST endpoint URL for a table.
// Supabase PostgREST is available at {supabase_url}/rest/v1/{table}.
func restURL(baseURL, table string) string {
	return baseURL + "/rest/v1/" + url.PathEscape(table)
}

// doRequest executes an HTTP request against the Supabase PostgREST API.
func (c *SupabaseConnector) doRequest(ctx context.Context, method, reqURL, apiKey string, body io.Reader, dest any) error {
	_, err := c.doRequestWithHeaders(ctx, method, reqURL, apiKey, body, dest, nil)
	return err
}

// doRequestWithHeaders is the core HTTP method. It accepts optional extra
// headers and returns the Content-Range response header (useful for exact
// count queries).
func (c *SupabaseConnector) doRequestWithHeaders(ctx context.Context, method, reqURL, apiKey string, body io.Reader, dest any, extraHeaders map[string]string) (contentRange string, err error) {
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("apikey", apiKey)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	// Request response as JSON.
	req.Header.Set("Accept", "application/json")

	// For mutations, request the rows back via Prefer header.
	if method == http.MethodPost || method == http.MethodPatch || method == http.MethodDelete {
		req.Header.Set("Prefer", "return=representation")
	}

	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return "", &connectors.TimeoutError{Message: fmt.Sprintf("Supabase API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return "", &connectors.CanceledError{Message: "Supabase API request canceled"}
		}
		return "", &connectors.ExternalError{Message: fmt.Sprintf("Supabase API request failed: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(resp.Header.Get("Retry-After"), defaultRetryAfter)
		return "", &connectors.RateLimitError{
			Message:    "Supabase API rate limit exceeded",
			RetryAfter: retryAfter,
		}
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return "", &connectors.AuthError{
			Message: "Supabase authentication failed — check that your API key is valid",
		}
	}
	if resp.StatusCode == http.StatusForbidden {
		return "", &connectors.AuthError{
			Message: "Supabase permission denied — the API key may lack access to this table (check RLS policies)",
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", mapSupabaseError(resp.StatusCode, respBody)
	}

	if dest != nil {
		if err := json.Unmarshal(respBody, dest); err != nil {
			return "", &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    "failed to decode Supabase PostgREST response",
			}
		}
	}

	return resp.Header.Get("Content-Range"), nil
}

// parseTotalFromContentRange extracts the total count from a PostgREST
// Content-Range header (e.g., "0-9/42" → 42). Returns -1 if unparseable.
func parseTotalFromContentRange(header string) int {
	slashIdx := strings.LastIndexByte(header, '/')
	if slashIdx < 0 || slashIdx+1 >= len(header) {
		return -1
	}
	total := header[slashIdx+1:]
	if total == "*" {
		return -1
	}
	n, err := strconv.Atoi(total)
	if err != nil {
		return -1
	}
	return n
}

