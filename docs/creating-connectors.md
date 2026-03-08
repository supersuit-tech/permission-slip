# Creating Connectors and Actions

This guide walks through adding a new connector (an integration with an external service) and adding actions to it. It uses the existing GitHub, Slack, PostgreSQL, Amadeus, Square, and Twilio connectors as reference implementations.

**Which reference to follow:** Browse the existing connectors in [`connectors/`](../connectors/) for reference implementations covering API key auth (GitHub, Notion), OAuth 2.0 (Google), basic auth (Jira), HTTP Basic Auth with form-encoded POSTs (Twilio), custom auth (Slack), JWT-based auth (DoorDash), and more. The Jira connector (`connectors/jira/`) is a good starting reference for basic auth with dynamic base URLs and SSRF-safe credential validation. The Shopify connector (`connectors/shopify/`) is a good reference for multi-step API flows (create_discount) and comprehensive parameter validation with allowlists. The Twilio connector (`connectors/twilio/`) is a good reference for HTTP Basic Auth, form-encoded write operations, separate read/write HTTP helpers (`doForm`/`doGet`), and using two different API base URLs (REST API + Lookup API). The Slack connector (`connectors/slack/`) is a good reference for custom Bearer token auth, shared response types across similar actions (`messages.go`), input validation helpers (`validateChannelID`, `validateLimit`), and response body size limits. The Notion connector (`connectors/notion/`) is a good reference for API-versioned services, optional JSON parameter fields (`json.RawMessage`), pagination support, and convenience helpers (auto-wrapping text as blocks). The DoorDash connector (`connectors/doordash/`) is a good reference for self-signed JWT auth (HS256 with base64-decoded secret), status enum validation, and `RequiresPaymentMethod` for financial actions. The Plaid connector (`connectors/plaid/`) is a good reference for custom body-based auth (client_id/secret in request body), sandbox/production environment switching via credentials, API version pinning, shared action types for DRY endpoints, and preventing credential leakage in HTTP helpers. The Datadog connector (`connectors/datadog/`) is a good reference for multi-region site routing via credentials, dual-key auth (API key + application key), and high-risk action classification (trigger_runbook). The PagerDuty connector (`connectors/pagerduty/`) is a good reference for incident lifecycle management (triggered → acknowledged → resolved), the `From` email header pattern for audit trails, and read-only vs. write action risk differentiation. The Airtable connector (`connectors/airtable/`) is a good reference for strict ID format validation (alphanumeric-only to prevent URL injection), batch operations with size limits, a generic `parseAndValidate` helper to reduce unmarshal/validate boilerplate, and formula-based search with detailed parameter descriptions for non-developer users. The Zapier connector (`connectors/zapier/`) is a good reference for webhook-based integrations with SSRF-safe URL prefix validation, redirect-disabled HTTP clients, and fire-and-forget vs. synchronous execution modes. The Make connector (`connectors/make/`) is a good reference for region-based multi-datacenter routing via credential allowlists (instead of arbitrary URLs), `url.Values` for safe query string construction, and structured response wrapping for better UX. The QuickBooks connector (`connectors/quickbooks/`) is a good reference for OAuth 2.0 with realm/company-scoped credentials, high-risk financial actions with explicit warnings, allowlist-based query injection prevention, and comprehensive manifest templates for progressive permission levels (read-only → medium-risk → high-risk). The Monday connector (`connectors/monday/`) is a good reference for GraphQL-only APIs (single POST endpoint), API key auth without Bearer prefix, numeric ID validation, stringified JSON column values, variable-based query parameterization (avoiding GraphQL injection), and splitting large manifests into a separate `manifest.go` file.

For architectural context, see [ADR-009: Connector Execution Architecture](adr/009-connector-execution-architecture.md).

---

## Overview

A **connector** represents an integration with an external service or database (e.g., GitHub, Slack, PostgreSQL). A connector owns shared configuration like HTTP clients, base URLs, and authentication helpers.

An **action** is a single operation within a connector (e.g., `github.create_issue`, `slack.send_message`). Each action has its own file, parameter struct, validation, and `Execute` method.

The architecture follows a two-level interface pattern:

```
Connector (shared state: HTTP client, base URL, auth)
  └── Action (parameter parsing, API call, response mapping)
```

### Key design principles

- **In-process Go**: Connectors compile into the binary. No plugins, sidecars, or external processes.
- **One action per file**: Adding an action means adding one file + one line of registration.
- **Plain `net/http` or `database/sql`**: HTTP connectors (GitHub, Slack) use plain net/http. Database connectors (MySQL) use database/sql with parameterized queries. No third-party SDKs beyond drivers.
- **Typed errors**: Actions return specific error types that map to HTTP status codes.
- **Credentials at execution time**: Decrypted from the vault only when an action runs, never cached.

---

## Adding a New Connector

This section walks through creating a connector from scratch. We'll use a hypothetical "Jira" connector as an example.

### Step 1: Create the package directory

```
connectors/jira/
```

### Step 2: Implement the connector struct

Create `connectors/jira/jira.go`. This file owns the shared HTTP client and helper methods that all actions in this connector will use.

```go
// Package jira implements the Jira connector for the Permission Slip
// connector execution layer. It uses the Jira REST API with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultTimeout = 30 * time.Second
)

// JiraConnector owns the shared HTTP client used by all Jira actions.
// Actions hold a pointer back to the connector to access these shared
// resources.
type JiraConnector struct {
	client *http.Client
}

// New creates a JiraConnector with sensible defaults (30s timeout).
func New() *JiraConnector {
	return &JiraConnector{
		client: &http.Client{Timeout: defaultTimeout},
	}
}

// newForTest creates a JiraConnector that points at a test server.
func newForTest(client *http.Client) *JiraConnector {
	return &JiraConnector{
		client: client,
	}
}

// ID returns "jira", matching the connectors.id in the database.
func (c *JiraConnector) ID() string { return "jira" }

// Actions returns the registered action handlers keyed by action_type.
func (c *JiraConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"jira.create_issue": &createIssueAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain the
// required fields for Jira API calls.
func (c *JiraConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	email, ok := creds.Get("email")
	if !ok || email == "" {
		return &connectors.ValidationError{Message: "missing required credential: email"}
	}
	token, ok := creds.Get("api_token")
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_token"}
	}
	return nil
}

// do is the shared request lifecycle for all Jira actions. It marshals
// reqBody as JSON, sends the request with auth headers, checks the response
// status, and unmarshals the response into respBody.
func (c *JiraConnector) do(ctx context.Context, creds connectors.Credentials, method, url string, reqBody, respBody any) error {
	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Jira Cloud uses Basic Auth with email + API token.
	email, _ := creds.Get("email")
	token, _ := creds.Get("api_token")
	req.SetBasicAuth(email, token)

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Jira API request timed out: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Jira API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mapJiraError(resp.StatusCode, respBytes)
	}

	if respBody != nil {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Jira response: %v", err)}
		}
	}
	return nil
}

// mapJiraError converts Jira API errors to typed connector errors.
func mapJiraError(statusCode int, body []byte) error {
	msg := string(body)

	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Jira API auth error (%d): %s", statusCode, msg)}
	case statusCode == http.StatusTooManyRequests:
		return &connectors.RateLimitError{Message: "Jira API rate limit exceeded"}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("Jira API error: %s", msg)}
	}
}
```

**Key patterns to follow:**

| Pattern | Why |
|---------|-----|
| `New()` constructor with sensible defaults | Production use |
| `newForTest()` accepting `*http.Client` | Enables `httptest.NewServer` in tests |
| Shared `do()` helper method | Eliminates duplication across actions (auth headers, error mapping, JSON marshaling) |
| `ID()` returns a string matching the DB `connectors.id` | Registry + DB validation at startup |
| `ValidateCredentials()` checks format only | Called before execution; doesn't hit the external API |

### Step 3: Add an action

Create `connectors/jira/create_issue.go`. Each action gets its own file.

```go
package jira

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createIssueAction implements connectors.Action for jira.create_issue.
type createIssueAction struct {
	conn *JiraConnector
}

// createIssueParams are the parameters parsed from ActionRequest.Parameters.
type createIssueParams struct {
	BaseURL    string `json:"base_url"`     // e.g., "https://mycompany.atlassian.net"
	ProjectKey string `json:"project_key"`  // e.g., "PROJ"
	Summary    string `json:"summary"`
	IssueType  string `json:"issue_type"`   // e.g., "Task", "Bug"
}

func (p *createIssueParams) validate() error {
	if p.BaseURL == "" {
		return &connectors.ValidationError{Message: "missing required parameter: base_url"}
	}
	if p.ProjectKey == "" {
		return &connectors.ValidationError{Message: "missing required parameter: project_key"}
	}
	if p.Summary == "" {
		return &connectors.ValidationError{Message: "missing required parameter: summary"}
	}
	if p.IssueType == "" {
		p.IssueType = "Task" // sensible default
	}
	return nil
}

// Execute creates a Jira issue and returns the created issue data.
func (a *createIssueAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	// 1. Parse and validate parameters
	var params createIssueParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// 2. Build the external API request body
	jiraBody := map[string]any{
		"fields": map[string]any{
			"project":   map[string]string{"key": params.ProjectKey},
			"summary":   params.Summary,
			"issuetype": map[string]string{"name": params.IssueType},
		},
	}

	// 3. Call the external API via the shared do() helper
	var jiraResp struct {
		ID   string `json:"id"`
		Key  string `json:"key"`
		Self string `json:"self"`
	}

	url := params.BaseURL + "/rest/api/3/issue"
	if err := a.conn.do(ctx, req.Credentials, "POST", url, jiraBody, &jiraResp); err != nil {
		return nil, err
	}

	// 4. Return the result
	return connectors.JSONResult(jiraResp)
}
```

**Every action follows this pattern:**

1. **Parse parameters** — `json.Unmarshal` into a typed struct
2. **Validate parameters** — Return `*connectors.ValidationError` for missing/invalid fields
3. **Call the external API** — Use the connector's shared `do()` / `doPost()` helper
4. **Return the result** — Use `connectors.JSONResult()` to wrap the response

### Step 4: Implement ManifestProvider

Add a `Manifest()` method to your connector. This returns a `ConnectorManifest` describing the connector's identity, actions, and required credentials. The server auto-seeds DB rows from this manifest on startup — no manual SQL or seed files needed.

```go
// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
func (c *JiraConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "jira",
		Name:        "Jira",
		Description: "Jira integration for project and issue management",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "jira.create_issue",
				Name:        "Create Issue",
				Description: "Create a new Jira issue",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["base_url", "project_key", "summary"],
					"properties": {
						"base_url": {
							"type": "string",
							"description": "Jira instance URL (e.g. https://mycompany.atlassian.net)"
						},
						"project_key": {
							"type": "string",
							"description": "Project key (e.g. PROJ)"
						},
						"summary": {
							"type": "string",
							"description": "Issue summary"
						},
						"issue_type": {
							"type": "string",
							"enum": ["Task", "Bug", "Story", "Epic"],
							"default": "Task",
							"description": "Issue type"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "jira", AuthType: "basic", InstructionsURL: "https://support.atlassian.com/atlassian-account/docs/manage-api-tokens-for-your-atlassian-account/"},
		},
	}
}
```

The `ParametersSchema` is a JSON Schema object that describes the action's parameters. The frontend uses this to:
- Show human-readable descriptions instead of raw parameter keys in the approval review modal
- Display type annotations (string, integer, boolean) for each parameter
- Show enum choices and default values for constrained parameters
- Mark required vs. optional parameters

Use `connectors.TrimIndent()` to keep inline JSON readable while stripping the source-level tab indentation from the string literal.

**Three tables auto-populated from the manifest:**

| Manifest Field | DB Table | Purpose |
|----------------|----------|---------|
| Top-level fields | `connectors` | One row per connector (id, name, description) |
| `actions[]` | `connector_actions` | One row per action (action_type, risk_level, optional parameters_schema, requires_payment_method) |
| `required_credentials[]` | `connector_required_credentials` | What credentials this connector needs (service, auth_type, instructions_url, oauth_provider, oauth_scopes) |

**Auth types:** `api_key`, `basic`, `custom`, `oauth2`

When using `oauth2`, the credential entry must include `oauth_provider` (e.g., `"github"`, `"google"`, `"microsoft"`, `"square"`) and optionally `oauth_scopes`. Built-in providers (`github`, `google`, `kroger`, `linkedin`, `meta`, `microsoft`, `salesforce`, `square`, `stripe`, `zoom`) are supported out of the box. External connectors can declare custom providers in the manifest's `oauth_providers` section (see below).

A connector can support multiple auth methods by declaring more than one entry in `RequiredCredentials`. For example, the Square connector supports both OAuth (recommended) and API key authentication. Use different `Service` names for each entry (e.g., `"square"` for OAuth, `"square_api_key"` for API key). The execution layer tries OAuth first and falls back to static credentials.

```go
// Example: OAuth2 credential in a manifest
RequiredCredentials: []connectors.ManifestCredential{
    {
        Service:       "google",
        AuthType:      "oauth2",
        OAuthProvider: "google",
        OAuthScopes:   []string{"https://www.googleapis.com/auth/gmail.send"},
    },
},
```

#### Declaring custom OAuth providers

External connectors that use OAuth providers not built into the platform (anything other than `github`, `google`, `kroger`, `linkedin`, `meta`, `microsoft`, `salesforce`, `square`, `stripe`, or `zoom`) must declare them in the manifest's `oauth_providers` section. The platform uses these URLs to drive the OAuth authorization flow.

```go
OAuthProviders: []connectors.ManifestOAuthProvider{
    {
        ID:           "salesforce",
        AuthorizeURL: "https://login.salesforce.com/services/oauth2/authorize",
        TokenURL:     "https://login.salesforce.com/services/oauth2/token",
        Scopes:       []string{"api", "refresh_token"},
    },
},
```

Requirements:
- `authorize_url` and `token_url` must use HTTPS
- Provider IDs must be unique within the manifest and must be lowercase alphanumeric with hyphens/underscores (1-63 chars)
- Any `oauth_provider` referenced in `required_credentials` must either be a built-in provider or declared in `oauth_providers`

**How it works at runtime:** On startup, the platform builds an OAuth Provider Registry (`oauth.Registry`). Built-in providers (Google, Microsoft) are registered first with endpoints and default scopes. Then, providers declared in connector manifests are merged in. The registry uses a priority system (BYOA > Manifest > BuiltIn) so that user-provided "bring your own app" credentials overlay the platform's built-in configuration. When a BYOA user registers credentials for a provider, only their client ID and secret are required — endpoints and scopes are inherited from the built-in or manifest definition.

**Risk levels:** `low`, `medium`, `high`

#### Payment-requiring actions

If an action involves a financial transaction (e.g., booking, purchasing), set `RequiresPaymentMethod: true` in the manifest action:

```go
{
    ActionType:            "travel.book_flight",
    Name:                  "Book Flight",
    Description:           "Book a flight itinerary",
    RiskLevel:             "high",
    RequiresPaymentMethod: true,
    ParametersSchema:      json.RawMessage(`{...}`),
},
```

When this flag is set:
- The API layer requires `payment_method_id` and `amount_cents` in the execute request
- Per-transaction and monthly spending limits are enforced before execution
- `req.Payment` is populated with a `*PaymentInfo` containing the Stripe token and card metadata
- The action implementation can use `req.Payment` to pass billing details to the external service
- A transaction record is created after successful execution for monthly spend tracking

### Step 5: Register in main.go

Add a line to `main.go`. Because the connector implements `ManifestProvider`, its DB rows will be auto-seeded on startup:

```go
import (
	// ...existing imports...
	jiraconnector "github.com/supersuit-tech/permission-slip-web/connectors/jira"
)

// In the startup section:
registry := connectors.NewRegistry()
registry.Register(ghconnector.New())
registry.Register(slack.New())
registry.Register(amadeus.New())
registry.Register(jiraconnector.New())  // ← add this
```

### Step 6: Register in the seed runner

Add the manifest to `cmd/seed/main.go` in the `seedConnectors` function's `builtins` slice:

```go
import (
	// ...existing imports...
	jiraconnector "github.com/supersuit-tech/permission-slip-web/connectors/jira"
)

// In seedConnectors():
builtins := []struct {
	manifest *connectors.ConnectorManifest
}{
	{ghconnector.New().Manifest()},
	{slackconnector.New().Manifest()},
	{jiraconnector.New().Manifest()},  // ← add this
}
```

### Step 7: Write tests

Create `connectors/jira/jira_test.go` for connector-level tests:

```go
package jira

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestJiraConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "jira" {
		t.Errorf("ID() = %q, want %q", got, "jira")
	}
}

func TestJiraConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	want := []string{"jira.create_issue"}
	for _, at := range want {
		if _, ok := actions[at]; !ok {
			t.Errorf("Actions() missing %q", at)
		}
	}
	if len(actions) != len(want) {
		t.Errorf("Actions() returned %d actions, want %d", len(actions), len(want))
	}
}

func TestJiraConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid credentials",
			creds:   connectors.NewCredentials(map[string]string{"email": "user@example.com", "api_token": "abc123"}),
			wantErr: false,
		},
		{
			name:    "missing email",
			creds:   connectors.NewCredentials(map[string]string{"api_token": "abc123"}),
			wantErr: true,
		},
		{
			name:    "missing api_token",
			creds:   connectors.NewCredentials(map[string]string{"email": "user@example.com"}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.ValidateCredentials(t.Context(), tt.creds)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJiraConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*JiraConnector)(nil)
}
```

Create `connectors/jira/helpers_test.go` for shared test utilities:

```go
package jira

import "github.com/supersuit-tech/permission-slip-web/connectors"

func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"email":     "user@example.com",
		"api_token": "test_token_123",
	})
}
```

Create `connectors/jira/create_issue_test.go` for action tests:

```go
package jira

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateIssue_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}

		// Verify Basic Auth is present
		user, pass, ok := r.BasicAuth()
		if !ok || user != "user@example.com" || pass != "test_token_123" {
			t.Errorf("BasicAuth = (%q, %q, %v)", user, pass, ok)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"id":   "10001",
			"key":  "PROJ-42",
			"self": "https://mycompany.atlassian.net/rest/api/3/issue/10001",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client())
	action := conn.Actions()["jira.create_issue"]

	params := map[string]string{
		"base_url":    srv.URL,
		"project_key": "PROJ",
		"summary":     "Test issue",
		"issue_type":  "Bug",
	}
	paramsJSON, _ := json.Marshal(params)

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.create_issue",
		Parameters:  paramsJSON,
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["key"] != "PROJ-42" {
		t.Errorf("key = %v, want PROJ-42", data["key"])
	}
}

func TestCreateIssue_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["jira.create_issue"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing base_url", params: `{"project_key":"PROJ","summary":"Bug"}`},
		{name: "missing project_key", params: `{"base_url":"https://x.atlassian.net","summary":"Bug"}`},
		{name: "missing summary", params: `{"base_url":"https://x.atlassian.net","project_key":"PROJ"}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "jira.create_issue",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("Execute() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}
```

**Testing patterns:**

- Use `httptest.NewServer` to mock the external API — no real API calls
- Use `newForTest()` to point the connector at the test server
- Test success path, missing/invalid parameters, API errors, auth failures, timeouts, and rate limits
- Use `t.Parallel()` on all tests
- Use typed error checks: `connectors.IsValidationError()`, `connectors.IsExternalError()`, etc.

---

## Adding an Action to an Existing Connector

Adding an action to an existing connector is simpler — the shared infrastructure already exists.

### Example: Adding `github.close_issue`

**1. Create the action file** — `connectors/github/close_issue.go`:

```go
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type closeIssueAction struct {
	conn *GitHubConnector
}

type closeIssueParams struct {
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`
	IssueNumber int    `json:"issue_number"`
}

func (p *closeIssueParams) validate() error {
	if p.Owner == "" {
		return &connectors.ValidationError{Message: "missing required parameter: owner"}
	}
	if p.Repo == "" {
		return &connectors.ValidationError{Message: "missing required parameter: repo"}
	}
	if p.IssueNumber <= 0 {
		return &connectors.ValidationError{Message: "missing or invalid required parameter: issue_number"}
	}
	return nil
}

func (a *closeIssueAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params closeIssueParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var ghResp struct {
		Number  int    `json:"number"`
		State   string `json:"state"`
		HTMLURL string `json:"html_url"`
	}

	path := fmt.Sprintf("/repos/%s/%s/issues/%d", url.PathEscape(params.Owner), url.PathEscape(params.Repo), params.IssueNumber)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPatch, path, map[string]string{"state": "closed"}, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
```

**2. Register in `Actions()`** — edit `connectors/github/github.go`:

```go
func (c *GitHubConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"github.create_issue": &createIssueAction{conn: c},
		"github.merge_pr":     &mergePRAction{conn: c},
		"github.close_issue":  &closeIssueAction{conn: c},  // ← add this
	}
}
```

**3. Add seed data** — edit `connectors/github/seed.go`:

```go
exec(ctx,
    `INSERT INTO connector_actions (connector_id, action_type, name, description, risk_level)
     VALUES ($1, $2, $3, $4, $5)`,
    "github", "github.close_issue", "Close Issue", "Close an open issue", "low")
```

**4. Write tests** — create `connectors/github/close_issue_test.go`

**5. Update the connector-level test** — add `"github.close_issue"` to the expected actions in `github_test.go`

That's it. No other files need to change — the registry, API handlers, and frontend all work automatically because the action type is registered in code and the database.

---

## Checklist

Use this checklist when adding a new connector or action.

### New Connector Checklist

- [ ] Create package directory: `connectors/<name>/`
- [ ] Implement connector struct with `ID()`, `Actions()`, `ValidateCredentials()`
- [ ] Implement `ManifestProvider` interface: `Manifest()` method returning `*ConnectorManifest`
- [ ] Add `ParametersSchema` (JSON Schema) for each action in the manifest
- [ ] Add shared `do()` / `doPost()` helper for HTTP lifecycle
- [ ] Add `New()` constructor and `newForTest()` for testing
- [ ] Implement at least one action (one file per action)
- [ ] Register connector in `main.go`: `registry.Register(yourconnector.New())`
- [ ] Add manifest to `cmd/seed/main.go` `seedConnectors` builtins slice
- [ ] Write connector-level tests (`ID`, `Actions`, `ValidateCredentials`, `Manifest`, interface check)
- [ ] Write action tests (success, missing params, API errors, auth failures, timeouts)
- [ ] Add `helpers_test.go` with `validCreds()` test helper
- [ ] Run `make test-backend` — all tests pass
- [ ] Run `make build` — compiles cleanly

### New Action Checklist (existing connector)

- [ ] Create action file: `connectors/<connector>/<action_name>.go`
- [ ] Define params struct with `validate()` method
- [ ] Implement `Execute` following the parse → validate → call → return pattern
- [ ] Register in connector's `Actions()` map
- [ ] Add action to connector's `Manifest()` return value with `ParametersSchema` (and `RequiresPaymentMethod: true` if applicable)
- [ ] Write tests: `connectors/<connector>/<action_name>_test.go`
- [ ] Update connector-level test's expected action list
- [ ] Run `make test-backend` — all tests pass
- [ ] Run `make build` — compiles cleanly

---

## Reference

### Core interfaces (`connectors/connector.go`)

```go
type Action interface {
	Execute(ctx context.Context, req ActionRequest) (*ActionResult, error)
}

type Connector interface {
	ID() string
	Actions() map[string]Action
	ValidateCredentials(ctx context.Context, creds Credentials) error
}

// ManifestProvider is optionally implemented by connectors that can
// describe their metadata declaratively. The server uses it to
// auto-seed DB rows on startup.
type ManifestProvider interface {
	Manifest() *ConnectorManifest
}

type ActionRequest struct {
	ActionType  string          // e.g., "github.create_issue"
	Parameters  json.RawMessage // validated against schema before reaching here
	Credentials Credentials     // decrypted at execution time; redacted in logs and JSON
	Payment     *PaymentInfo    // non-nil when the action requires a payment method
}

// PaymentInfo is set when the action declares RequiresPaymentMethod: true.
// Raw card data is never included — only the Stripe token and safe metadata.
type PaymentInfo struct {
	StripePaymentMethodID string // Stripe payment method token (e.g. "pm_...")
	Brand                 string // card brand (e.g. "visa")
	Last4                 string // last 4 digits of the card
	AmountCents           int    // authorized transaction amount in cents
}

type ActionResult struct {
	Data json.RawMessage // service-specific response payload
}
```

### Error types (`connectors/errors.go`)

| Error Type | HTTP Status | When to use |
|------------|-------------|-------------|
| `*ValidationError` | 400 | Missing/invalid parameters or credentials |
| `*AuthError` | 502 | External service rejected credentials |
| `*ExternalError` | 502 | External service returned a non-success response |
| `*RateLimitError` | 429 | External service rate-limited the request (include `RetryAfter`) |
| `*TimeoutError` | 504 | External service didn't respond in time |
| `*OAuthRefreshError` | 401 | OAuth token refresh failed — user must re-authorize the provider |
| `*PaymentError` | 400/403 | Payment method missing, not found, or limit exceeded (see codes below) |

**Payment error codes** (`PaymentError.Code`):

| Code | HTTP Status | Meaning |
|------|-------------|---------|
| `PaymentErrMissing` | 400 | `payment_method_id` not provided for an action that requires payment |
| `PaymentErrNotFound` | 400 | Payment method does not exist or doesn't belong to the user |
| `PaymentErrAmountRequired` | 400 | `amount_cents` is required but not provided |
| `PaymentErrInvalidAmount` | 400 | `amount_cents` is negative or otherwise invalid |
| `PaymentErrPerTxLimit` | 403 | Amount exceeds the payment method's per-transaction limit |
| `PaymentErrMonthlyLimit` | 403 | Amount would exceed the payment method's monthly spending limit |

Limit errors (`PaymentErrPerTxLimit`, `PaymentErrMonthlyLimit`) include a `Details` map with structured information (e.g., `limit`, `current_spend`, `requested_amount`) to help agents display actionable messages.

### Helper functions (`connectors/helpers.go`)

| Function | Purpose |
|----------|---------|
| `connectors.JSONResult(v)` | Marshals any value into an `*ActionResult` |
| `connectors.IsTimeout(err)` | Checks if an error is a timeout (context deadline or net.Error) |
| `connectors.ParseRetryAfter(val, fallback)` | Parses a `Retry-After` header value into `time.Duration` |

### Credentials (`connectors/credentials.go`)

```go
// Create credentials (done by the execution layer, not by actions)
creds := connectors.NewCredentials(map[string]string{"api_key": "ghp_..."})

// Read a credential value (done by actions)
key, ok := creds.Get("api_key")
```

Credentials automatically redact themselves in `String()`, `GoString()`, and `MarshalJSON()` to prevent accidental logging or serialization.

### Database schema

```sql
-- One row per connector
CREATE TABLE connectors (
    id          text PRIMARY KEY,
    name        text NOT NULL,
    description text,
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- One row per action
CREATE TABLE connector_actions (
    connector_id              text NOT NULL REFERENCES connectors(id),
    action_type               text NOT NULL,         -- "github.create_issue"
    name                      text NOT NULL,         -- "Create Issue"
    description               text,
    risk_level                text,                  -- 'low' | 'medium' | 'high'
    parameters_schema         jsonb,                 -- optional JSON Schema
    requires_payment_method   boolean NOT NULL DEFAULT false,
    PRIMARY KEY (connector_id, action_type)
);

-- What credentials a connector needs
CREATE TABLE connector_required_credentials (
    connector_id    text NOT NULL REFERENCES connectors(id),
    service         text NOT NULL,              -- credential service identifier
    auth_type       text NOT NULL,              -- 'api_key' | 'basic' | 'custom' | 'oauth2'
    instructions_url text,                      -- optional URL for credential setup docs
    oauth_provider  text,                       -- required when auth_type = 'oauth2'
    oauth_scopes    text[] DEFAULT '{}',        -- OAuth scopes when auth_type = 'oauth2'
    PRIMARY KEY (connector_id, service)
);
```

### Action type naming convention

Action types follow the pattern `<connector_id>.<action_name>`:

- `github.create_issue`
- `github.merge_pr`
- `slack.send_message`
- `slack.create_channel`
- `slack.schedule_message`
- `slack.upload_file`

The registry uses the part before the first `.` to route to the correct connector.

### File structure

```
connectors/
├── connector.go              # Connector, Action, and ManifestProvider interfaces
├── manifest.go               # ConnectorManifest, ManifestAction, ManifestCredential types
├── credentials.go            # Credentials value type (redacts on log/JSON)
├── errors.go                 # Typed error types
├── helpers.go                # JSONResult, IsTimeout, ParseRetryAfter
├── registry.go               # Registry (maps connector IDs → implementations)
├── github/
│   ├── github.go             # GitHubConnector struct, New(), Manifest(), do(), ValidateCredentials()
│   ├── response.go           # HTTP status → typed error mapping
│   ├── create_issue.go       # github.create_issue action
│   ├── merge_pr.go           # github.merge_pr action
│   ├── helpers_test.go       # validCreds() test helper
│   ├── github_test.go        # Connector-level tests
│   ├── create_issue_test.go  # Action tests
│   └── merge_pr_test.go      # Action tests
├── mysql/
│   ├── mysql.go              # MySQLConnector struct, New(), Manifest(), openConn(), shared helpers
│   ├── query.go              # mysql.query action (read-only, parameterized SELECT)
│   ├── insert.go             # mysql.insert action (parameterized INSERT with batch limits)
│   ├── update.go             # mysql.update action (parameterized UPDATE, WHERE required)
│   ├── delete.go             # mysql.delete action (parameterized DELETE, WHERE required)
│   ├── helpers_test.go       # validCreds(), newTestConnector() with sqlmock
│   ├── mysql_test.go         # Connector-level tests
│   └── *_test.go             # Per-action tests
├── jira/
│   ├── jira.go              # JiraConnector struct, New(), do(), apiBase(), ValidateCredentials()
│   ├── manifest.go           # Manifest() with 6 action schemas and templates
│   ├── response.go           # HTTP status → typed error mapping
│   ├── adf.go                # plainTextToADF() — shared ADF conversion
│   ├── create_issue.go       # jira.create_issue action
│   ├── update_issue.go       # jira.update_issue action
│   ├── transition_issue.go   # jira.transition_issue action
│   ├── add_comment.go        # jira.add_comment action
│   ├── assign_issue.go       # jira.assign_issue action
│   ├── search.go             # jira.search action (JQL)
│   ├── README.md             # Connector documentation
│   ├── helpers_test.go       # validCreds() test helper
│   └── *_test.go             # Per-action + connector + response tests
├── google/
│   ├── google.go             # GoogleConnector struct, New(), Actions(), doJSON(), OAuth2 auth
│   ├── manifest.go           # Manifest() with 22 action schemas and 31+ templates
│   ├── send_email.go         # google.send_email action (RFC 2822 + base64url)
│   ├── list_emails.go        # google.list_emails action (list + metadata fetch)
│   ├── create_calendar_event.go  # google.create_calendar_event action
│   ├── list_calendar_events.go   # google.list_calendar_events action
│   ├── create_presentation.go    # google.create_presentation action (Slides API)
│   ├── get_presentation.go       # google.get_presentation action (Slides API)
│   ├── add_slide.go              # google.add_slide action (batchUpdate)
│   ├── slides_helpers.go         # Shared helpers for Slides actions
│   ├── sheets_read.go        # google.sheets_read_range action
│   ├── sheets_write.go       # google.sheets_write_range action
│   ├── sheets_append.go      # google.sheets_append_rows action
│   ├── sheets_list.go        # google.sheets_list_sheets action
│   ├── sheets_helpers.go     # Shared validation (row/cell limits, ragged row check)
│   ├── send_chat_message.go  # google.send_chat_message action (Google Chat API)
│   ├── list_chat_spaces.go   # google.list_chat_spaces action (Google Chat API)
│   ├── create_meeting.go     # google.create_meeting action (Calendar + Meet link)
│   ├── calendar_helpers.go   # Shared calendar validation (time range, attendees)
│   ├── README.md             # Connector documentation
│   └── ...tests...
├── slack/
│   ├── slack.go              # SlackConnector struct, New(), Actions(), doPost(), shared validators
│   ├── manifest.go           # Manifest() — action schemas, templates, credentials
│   ├── messages.go           # Shared message types (slackMessage, messageSummary, messagesResponse)
│   ├── send_message.go       # slack.send_message action
│   ├── create_channel.go     # slack.create_channel action
│   ├── list_channels.go      # slack.list_channels action
│   ├── read_channel_messages.go  # slack.read_channel_messages action
│   ├── read_thread.go        # slack.read_thread action
│   ├── schedule_message.go   # slack.schedule_message action
│   ├── set_topic.go          # slack.set_topic action
│   ├── invite_to_channel.go  # slack.invite_to_channel action
│   ├── upload_file.go        # slack.upload_file action (v2 upload flow)
│   ├── add_reaction.go       # slack.add_reaction action
│   ├── README.md             # Connector documentation
│   └── ...tests...
├── kroger/
│   ├── kroger.go              # KrogerConnector struct, New(), do(), OAuth2 bearer auth
│   ├── manifest.go            # Manifest() with 4 action schemas and 6 templates
│   ├── response.go            # HTTP status → typed error mapping (truncates large bodies)
│   ├── search_products.go     # kroger.search_products — search by keyword with location pricing
│   ├── get_product.go         # kroger.get_product — product details by UPC
│   ├── search_locations.go    # kroger.search_locations — find stores by zip/coordinates
│   ├── add_to_cart.go         # kroger.add_to_cart — add items to user's cart
│   ├── README.md              # Connector documentation
│   └── ...tests...
├── zoom/
│   ├── zoom.go                # ZoomConnector struct, New(), doJSON(), OAuth2 auth, error mapping
│   ├── manifest.go            # Manifest() with 7 action schemas and 8 templates
│   ├── list_meetings.go       # zoom.list_meetings action
│   ├── create_meeting.go      # zoom.create_meeting action
│   ├── get_meeting.go         # zoom.get_meeting action
│   ├── update_meeting.go      # zoom.update_meeting action
│   ├── delete_meeting.go      # zoom.delete_meeting action
│   ├── list_recordings.go     # zoom.list_recordings action
│   ├── get_meeting_participants.go  # zoom.get_meeting_participants action
│   └── ...tests...
├── shopify/
│   ├── shopify.go            # ShopifyConnector struct, New(), do(), dynamic base URL from shop_domain
│   ├── manifest.go           # Manifest() with 6 action schemas and 7 templates
│   ├── response.go           # HTTP status → typed error mapping (handles 3 Shopify error formats)
│   ├── get_orders.go         # shopify.get_orders — list/filter orders with query params
│   ├── get_order.go          # shopify.get_order — single order by ID
│   ├── update_order.go       # shopify.update_order — partial update via PUT
│   ├── create_product.go     # shopify.create_product — create with optional variants
│   ├── update_inventory.go   # shopify.update_inventory — relative inventory adjustment
│   ├── create_discount.go    # shopify.create_discount — two-step: price rule → discount code
│   ├── README.md             # Connector documentation
│   ├── helpers_test.go       # validCreds() test helper
│   └── *_test.go             # Per-action + connector + response tests
├── twilio/
│   ├── twilio.go             # TwilioConnector struct, New(), Manifest(), doForm(), doGet(), Basic Auth
│   ├── response.go           # HTTP status → typed error mapping (Twilio error codes + more_info URL)
│   ├── send_sms.go           # twilio.send_sms action (SMS/MMS with optional MediaUrl)
│   ├── send_whatsapp.go      # twilio.send_whatsapp action (auto-prefixes whatsapp: on numbers)
│   ├── initiate_call.go      # twilio.initiate_call action (TwiML-based voice calls)
│   ├── get_message.go        # twilio.get_message action (SM/MM SID validation)
│   ├── get_call.go           # twilio.get_call action (CA SID validation)
│   ├── lookup_phone.go       # twilio.lookup_phone action (Lookup API v2)
│   ├── README.md             # Connector documentation
│   ├── helpers_test.go       # validCreds() test helper
│   └── *_test.go             # Per-action + connector + response tests
├── doordash/
│   ├── doordash.go           # DoorDashConnector struct, New(), JWT auth (HS256 + base64 secret), do()
│   ├── manifest.go           # Manifest() with 5 action schemas and templates
│   ├── response.go           # HTTP status → typed error mapping
│   ├── create_delivery.go    # doordash.create_delivery action (high risk, RequiresPaymentMethod)
│   ├── get_quote.go          # doordash.get_quote action
│   ├── get_delivery.go       # doordash.get_delivery action
│   ├── cancel_delivery.go    # doordash.cancel_delivery action
│   ├── list_deliveries.go    # doordash.list_deliveries action (status enum validation)
│   ├── README.md             # Connector documentation
│   ├── helpers_test.go       # validCreds() test helper
│   └── *_test.go             # Per-action + connector + response tests
└── expedia/
    ├── expedia.go            # ExpediaConnector struct, New(), SHA-512 signature auth, do()
    ├── manifest.go           # Manifest() with 6 action schemas and templates
    ├── response.go           # HTTP status → typed error mapping
    ├── README.md             # Connector documentation
    ├── helpers_test.go       # validCreds() test helper
    ├── expedia_test.go       # Connector and do() lifecycle tests
    └── response_test.go      # checkResponse() error mapping tests
```

### Execution flow

```
Agent request → API validates auth → Registry.GetAction(actionType)
  → Decrypt credentials from Vault → connector.ValidateCredentials()
  → If requires_payment_method: validate payment method + check limits
  → action.Execute(ctx, ActionRequest{Payment: &PaymentInfo{...}})
  → External API call → ActionResult or typed error
  → If success + payment: record transaction against monthly spend
  → HTTP response to agent
```

The API layer handles credential decryption, validation orchestration, payment method resolution, and error-to-HTTP mapping. Action implementations only need to focus on calling the external API and returning results or typed errors.

**Payment flow details:** When an action declares `RequiresPaymentMethod: true`, the execution layer validates the payment method and checks per-transaction and monthly limits *before* calling `Execute`. The `Payment` field on `ActionRequest` is populated only for payment-requiring actions. Transactions are recorded *after* successful execution to avoid charging users for failed actions.
