package sendgrid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// sendTransactionalEmailAction implements connectors.Action for
// sendgrid.send_transactional_email. It sends a single transactional email
// via the SendGrid v3 POST /mail/send endpoint.
type sendTransactionalEmailAction struct {
	conn *SendGridConnector
}

type sendTransactionalEmailParams struct {
	To          string `json:"to"`
	ToName      string `json:"to_name"`
	From        string `json:"from"`
	FromName    string `json:"from_name"`
	Subject     string `json:"subject"`
	HTMLContent  string `json:"html_content"`
	PlainContent string `json:"plain_content"`
	TemplateID  string `json:"template_id"`
	// DynamicTemplateData is arbitrary key/value pairs passed to a dynamic template.
	DynamicTemplateData map[string]any `json:"dynamic_template_data"`
	ReplyTo             string         `json:"reply_to"`
	// CC and BCC support common transactional patterns (e.g. CC account managers,
	// BCC compliance inboxes). Each is a comma-separated list of email addresses.
	CC  []string `json:"cc"`
	BCC []string `json:"bcc"`
	// Categories are freeform labels that appear in SendGrid's activity/stats UI,
	// useful for filtering and analytics (e.g. ["welcome", "onboarding"]).
	Categories []string `json:"categories"`
}

func (p *sendTransactionalEmailParams) validate() error {
	if p.To == "" {
		return &connectors.ValidationError{Message: "missing required parameter: to"}
	}
	if !emailPattern.MatchString(p.To) {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid recipient email: %q", p.To)}
	}
	if p.From == "" {
		return &connectors.ValidationError{Message: "missing required parameter: from"}
	}
	if !emailPattern.MatchString(p.From) {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid sender email: %q", p.From)}
	}
	if p.Subject == "" && p.TemplateID == "" {
		return &connectors.ValidationError{Message: "subject is required when template_id is not provided"}
	}
	if p.HTMLContent == "" && p.PlainContent == "" && p.TemplateID == "" {
		return &connectors.ValidationError{Message: "at least one of html_content, plain_content, or template_id is required"}
	}
	// RFC 2822 §2.1.1 limits header lines to 998 characters.
	if len(p.Subject) > 998 {
		return &connectors.ValidationError{Message: "subject must be 998 characters or fewer (RFC 2822 limit)"}
	}
	if p.ReplyTo != "" && !emailPattern.MatchString(p.ReplyTo) {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid reply_to email: %q", p.ReplyTo)}
	}
	if len(p.DynamicTemplateData) > 0 && p.TemplateID == "" {
		return &connectors.ValidationError{Message: "dynamic_template_data requires template_id to be set"}
	}
	// SendGrid limits total recipients (to + cc + bcc) to 1000 per send.
	// We cap CC and BCC independently to prevent bulk-sending abuse through
	// the connector.
	if len(p.CC) > 100 {
		return &connectors.ValidationError{Message: "cc must contain at most 100 addresses"}
	}
	if len(p.BCC) > 100 {
		return &connectors.ValidationError{Message: "bcc must contain at most 100 addresses"}
	}
	if err := validateEmailAddresses("cc", p.CC); err != nil {
		return err
	}
	if err := validateEmailAddresses("bcc", p.BCC); err != nil {
		return err
	}
	if len(p.Categories) > 10 {
		return &connectors.ValidationError{Message: "categories must have at most 10 items (SendGrid limit)"}
	}
	for _, cat := range p.Categories {
		if len(cat) > 255 {
			return &connectors.ValidationError{Message: fmt.Sprintf("category name %q exceeds 255 character limit", cat[:40]+"...")}
		}
	}
	return nil
}

// Execute sends a transactional email via POST /mail/send.
func (a *sendTransactionalEmailAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendTransactionalEmailParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	toAddr := map[string]string{"email": params.To}
	if params.ToName != "" {
		toAddr["name"] = params.ToName
	}

	personalization := map[string]any{
		"to": []map[string]string{toAddr},
	}
	if len(params.CC) > 0 {
		ccList := make([]map[string]string, len(params.CC))
		for i, addr := range params.CC {
			ccList[i] = map[string]string{"email": addr}
		}
		personalization["cc"] = ccList
	}
	if len(params.BCC) > 0 {
		bccList := make([]map[string]string, len(params.BCC))
		for i, addr := range params.BCC {
			bccList[i] = map[string]string{"email": addr}
		}
		personalization["bcc"] = bccList
	}

	fromAddr := map[string]string{"email": params.From}
	if params.FromName != "" {
		fromAddr["name"] = params.FromName
	}

	body := map[string]any{
		"personalizations": []map[string]any{personalization},
		"from":             fromAddr,
	}

	// Subject and content are optional when a dynamic template is used.
	if params.Subject != "" {
		body["subject"] = params.Subject
	}

	var content []map[string]string
	if params.PlainContent != "" {
		content = append(content, map[string]string{"type": "text/plain", "value": params.PlainContent})
	}
	if params.HTMLContent != "" {
		content = append(content, map[string]string{"type": "text/html", "value": params.HTMLContent})
	}
	if len(content) > 0 {
		body["content"] = content
	}

	if params.TemplateID != "" {
		body["template_id"] = params.TemplateID
		if len(params.DynamicTemplateData) > 0 {
			// Inject dynamic data into the first personalization.
			perz := body["personalizations"].([]map[string]any)
			perz[0]["dynamic_template_data"] = params.DynamicTemplateData
			body["personalizations"] = perz
		}
	}

	if params.ReplyTo != "" {
		body["reply_to"] = map[string]string{"email": params.ReplyTo}
	}

	if len(params.Categories) > 0 {
		body["categories"] = params.Categories
	}

	// POST /mail/send returns 202 Accepted with an empty body on success.
	// We capture X-Message-Id from the response headers — this is SendGrid's
	// per-message identifier, useful for delivery troubleshooting via the
	// Activity Feed or email.
	messageID, err := a.conn.doJSONCapturingHeader(ctx, req.Credentials, http.MethodPost, "/mail/send", body, nil, "X-Message-Id")
	if err != nil {
		return nil, err
	}

	// SendGrid responds 202 Accepted — the message is queued, not yet delivered.
	result := map[string]any{
		"status": "accepted",
		"to":     params.To,
	}
	if messageID != "" {
		result["message_id"] = messageID
	}

	return connectors.JSONResult(result)
}
