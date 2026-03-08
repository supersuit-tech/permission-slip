// Package sendgrid implements the SendGrid connector for the Permission Slip
// connector execution layer. It uses the SendGrid v3 REST API with plain
// net/http (no third-party SDK) to keep the dependency footprint minimal.
package sendgrid

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
	defaultBaseURL = "https://api.sendgrid.com/v3"
	defaultTimeout = 30 * time.Second

	credKeyAPIKey = "api_key"

	// credKeyAccessToken is the credential key for OAuth2 access tokens,
	// set by the OAuth credential resolution path.
	credKeyAccessToken = "access_token"

	// maxResponseBody is the maximum response body size we'll read from SendGrid.
	maxResponseBody = 1 << 20 // 1 MiB

	// minAPIKeyLen is the minimum length for a SendGrid API key.
	// SendGrid API keys are typically prefixed with "SG." and are fairly long.
	minAPIKeyLen = 10
)

// SendGridConnector owns the shared HTTP client and base URL used by all
// SendGrid actions. Actions hold a pointer back to the connector to access
// these shared resources.
type SendGridConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a SendGridConnector with sensible defaults (30s timeout,
// production SendGrid API base URL).
func New() *SendGridConnector {
	return &SendGridConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a SendGridConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *SendGridConnector {
	return &SendGridConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "sendgrid", matching the connectors.id in the database.
func (c *SendGridConnector) ID() string { return "sendgrid" }

// Actions returns the registered action handlers keyed by action_type.
func (c *SendGridConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"sendgrid.send_campaign":      &sendCampaignAction{conn: c},
		"sendgrid.schedule_campaign":  &scheduleCampaignAction{conn: c},
		"sendgrid.add_to_list":        &addToListAction{conn: c},
		"sendgrid.remove_from_list":   &removeFromListAction{conn: c},
		"sendgrid.create_template":    &createTemplateAction{conn: c},
		"sendgrid.get_campaign_stats": &getCampaignStatsAction{conn: c},
		"sendgrid.list_segments":      &listSegmentsAction{conn: c},
		"sendgrid.list_senders":       &listSendersAction{conn: c},
		"sendgrid.list_lists":               &listListsAction{conn: c},
		"sendgrid.send_transactional_email": &sendTransactionalEmailAction{conn: c},
		"sendgrid.create_contact":           &createContactAction{conn: c},
		"sendgrid.get_bounces":              &getBouncesAction{conn: c},
		"sendgrid.get_suppressions":         &getSuppressionsAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty access token. Accepts either an OAuth2 access_token or an
// api_key — both are used as Bearer tokens against the SendGrid v3 API.
func (c *SendGridConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	if token, ok := creds.Get(credKeyAccessToken); ok && token != "" {
		return nil
	}
	key, ok := creds.Get(credKeyAPIKey)
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_key or access_token (OAuth)"}
	}
	if len(key) < minAPIKeyLen {
		return &connectors.ValidationError{Message: fmt.Sprintf("api_key is too short (minimum %d characters)", minAPIKeyLen)}
	}
	return nil
}

// doJSON sends a JSON-encoded request to the SendGrid API and decodes
// the response. SendGrid's v3 API uses application/json throughout.
// It accepts either an OAuth2 access_token or an api_key as the Bearer token.
func (c *SendGridConnector) doJSON(ctx context.Context, creds connectors.Credentials, method, path string, body any, respBody any) error {
	token, _ := creds.Get(credKeyAccessToken)
	if token == "" {
		token, _ = creds.Get(credKeyAPIKey)
	}
	if token == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_key or access_token (OAuth)"}
	}

	reqURL := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("marshaling request body: %v", err)}
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("creating request: %v", err)}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("SendGrid API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "SendGrid API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("SendGrid API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing SendGrid response: %v", err)}
		}
	}
	return nil
}

// doJSONCapturingHeader is like doJSON but also returns the value of a single
// named response header. Useful for endpoints like POST /mail/send that return
// an empty body with useful metadata only in headers (e.g. X-Message-Id).
func (c *SendGridConnector) doJSONCapturingHeader(ctx context.Context, creds connectors.Credentials, method, path string, body any, respBody any, headerName string) (string, error) {
	token, _ := creds.Get(credKeyAccessToken)
	if token == "" {
		token, _ = creds.Get(credKeyAPIKey)
	}
	if token == "" {
		return "", &connectors.ValidationError{Message: "missing required credential: api_key or access_token (OAuth)"}
	}

	reqURL := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return "", &connectors.ExternalError{Message: fmt.Sprintf("marshaling request body: %v", err)}
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return "", &connectors.ExternalError{Message: fmt.Sprintf("creating request: %v", err)}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return "", &connectors.TimeoutError{Message: fmt.Sprintf("SendGrid API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return "", &connectors.TimeoutError{Message: "SendGrid API request canceled"}
		}
		return "", &connectors.ExternalError{Message: fmt.Sprintf("SendGrid API request failed: %v", err)}
	}
	defer resp.Body.Close()

	headerValue := resp.Header.Get(headerName)

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return "", &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return "", err
	}

	if respBody != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return "", &connectors.ExternalError{Message: fmt.Sprintf("parsing SendGrid response: %v", err)}
		}
	}
	return headerValue, nil
}
