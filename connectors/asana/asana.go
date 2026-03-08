// Package asana implements the Asana connector for the Permission Slip
// connector execution layer. It uses the Asana REST API with plain net/http
// and personal access tokens for authentication.
package asana

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
	defaultBaseURL = "https://app.asana.com/api/1.0"
	defaultTimeout = 30 * time.Second
	// maxResponseBytes caps the response body we'll read from Asana APIs.
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB
)

// AsanaConnector owns the shared HTTP client and base URL used by all
// Asana actions. Actions hold a pointer back to the connector to access
// these shared resources.
type AsanaConnector struct {
	client  *http.Client
	baseURL string
}

// New creates an AsanaConnector with sensible defaults (30s timeout,
// https://app.asana.com/api/1.0 base URL).
func New() *AsanaConnector {
	return &AsanaConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates an AsanaConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *AsanaConnector {
	return &AsanaConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "asana", matching the connectors.id in the database.
func (c *AsanaConnector) ID() string { return "asana" }

// Actions returns the registered action handlers keyed by action_type.
func (c *AsanaConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"asana.create_task":     &createTaskAction{conn: c},
		"asana.update_task":     &updateTaskAction{conn: c},
		"asana.add_comment":     &addCommentAction{conn: c},
		"asana.complete_task":   &completeTaskAction{conn: c},
		"asana.create_subtask":  &createSubtaskAction{conn: c},
		"asana.search_tasks":    &searchTasksAction{conn: c},
		"asana.list_workspaces": &listWorkspacesAction{conn: c},
		"asana.list_projects":   &listProjectsAction{conn: c},
		"asana.create_project":  &createProjectAction{conn: c},
		"asana.delete_task":     &deleteTaskAction{conn: c},
		"asana.list_sections":   &listSectionsAction{conn: c},
		"asana.create_section":  &createSectionAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty api_key (personal access token).
func (c *AsanaConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	key, ok := creds.Get("api_key")
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_key"}
	}
	return nil
}

// asanaEnvelope is the standard Asana API response wrapper.
type asanaEnvelope struct {
	Data json.RawMessage `json:"data"`
}

// execRequest is the shared HTTP lifecycle: sets auth, sends the request,
// reads the (size-limited) response body, and maps HTTP errors to typed
// connector errors. Returns the raw response bytes on success.
func (c *AsanaConnector) execRequest(ctx context.Context, creds connectors.Credentials, req *http.Request) ([]byte, error) {
	key, ok := creds.Get("api_key")
	if !ok || key == "" {
		return nil, &connectors.ValidationError{Message: "api_key credential is missing or empty"}
	}
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("Asana API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return nil, &connectors.TimeoutError{Message: "Asana API request canceled"}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("Asana API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return nil, err
	}

	return respBytes, nil
}

// do is the shared request lifecycle for all Asana actions. It marshals
// reqBody as JSON (wrapping in {"data": ...}), sends the request with auth
// headers, checks the response status, unwraps the {"data": ...} envelope,
// and unmarshals into respBody.
func (c *AsanaConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, respBody interface{}) error {
	var body io.Reader
	if reqBody != nil {
		// Asana expects request bodies wrapped in {"data": ...}
		envelope := map[string]interface{}{"data": reqBody}
		payload, err := json.Marshal(envelope)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	respBytes, err := c.execRequest(ctx, creds, req)
	if err != nil {
		return err
	}

	// Unwrap the {"data": ...} envelope if the caller wants the response.
	if respBody != nil {
		var envelope asanaEnvelope
		if err := json.Unmarshal(respBytes, &envelope); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Asana response envelope: %v", err)}
		}
		if err := json.Unmarshal(envelope.Data, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Asana response data: %v", err)}
		}
	}
	return nil
}

// doRaw is like do but skips the request body envelope wrapping (for GET
// requests that don't send a body) and returns the raw unwrapped data.
func (c *AsanaConnector) doRaw(ctx context.Context, creds connectors.Credentials, method, fullURL string, respBody interface{}) error {
	req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	respBytes, err := c.execRequest(ctx, creds, req)
	if err != nil {
		return err
	}

	if respBody != nil {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Asana response: %v", err)}
		}
	}
	return nil
}
