package zapier

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// triggerWebhookAction implements connectors.Action for zapier.trigger_webhook.
// It sends a JSON payload to a Zapier webhook URL to trigger a Zap.
type triggerWebhookAction struct {
	conn *ZapierConnector
}

// triggerWebhookParams is the user-facing parameter schema.
type triggerWebhookParams struct {
	Payload         json.RawMessage `json:"payload"`
	WaitForResponse bool            `json:"wait_for_response"`
}

func (p *triggerWebhookParams) validate() error {
	if len(p.Payload) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: payload"}
	}
	// Verify payload is valid JSON.
	var raw json.RawMessage
	if err := json.Unmarshal(p.Payload, &raw); err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("payload must be valid JSON: %v", err)}
	}
	return nil
}

// Execute sends a POST request to the Zapier webhook URL with the given payload.
func (a *triggerWebhookAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params triggerWebhookParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	webhookURL, ok := req.Credentials.Get(credKeyWebhookURL)
	if !ok || webhookURL == "" {
		return nil, &connectors.ValidationError{Message: "webhook_url credential is missing or empty"}
	}
	// Defense-in-depth: re-validate the URL prefix at execution time,
	// not just in ValidateCredentials. Prevents SSRF if validation is
	// somehow bypassed. Skipped in tests where we use local test servers.
	if !a.conn.skipURLValidation && !strings.HasPrefix(webhookURL, webhookURLPrefix) {
		return nil, &connectors.ValidationError{
			Message: fmt.Sprintf("webhook_url must start with %q", webhookURLPrefix),
		}
	}

	// Unmarshal the payload to send as the POST body.
	var payload any
	if err := json.Unmarshal(params.Payload, &payload); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid payload JSON: %v", err)}
	}

	respBody, statusCode, err := a.conn.doPost(ctx, webhookURL, payload)
	if err != nil {
		return nil, err
	}

	// Zapier returns various status codes:
	// - 200: success (with or without response data)
	// - 410: webhook is no longer active (Zap was deleted or turned off)
	if statusCode == http.StatusGone {
		return nil, &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    "Zapier webhook is no longer active — the Zap may have been deleted or turned off",
		}
	}

	if statusCode < 200 || statusCode >= 300 {
		msg := fmt.Sprintf("Zapier webhook returned HTTP %d", statusCode)
		if len(respBody) > 0 {
			// Include truncated body for debugging, capping at 512 bytes
			// to avoid leaking large payloads.
			body := string(respBody)
			if len(body) > 512 {
				body = body[:512] + "..."
			}
			msg += ": " + body
		}
		return nil, &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    msg,
		}
	}

	result := map[string]any{
		"status":      "triggered",
		"http_status": statusCode,
		"synchronous": params.WaitForResponse,
	}

	// Always include response body when present — even in fire-and-forget
	// mode, Zapier returns a confirmation that helps users verify the
	// webhook was received.
	if len(respBody) > 0 {
		var respData any
		if err := json.Unmarshal(respBody, &respData); err == nil {
			result["response"] = respData
		} else {
			result["response"] = string(respBody)
		}
	}

	return connectors.JSONResult(result)
}
