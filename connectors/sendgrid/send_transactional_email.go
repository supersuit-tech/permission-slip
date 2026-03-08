package sendgrid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
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
	HTMLContent string `json:"html_content"`
	PlainContent string `json:"plain_content"`
	TemplateID  string `json:"template_id"`
	// DynamicTemplateData is arbitrary key/value pairs passed to a dynamic template.
	DynamicTemplateData map[string]any `json:"dynamic_template_data"`
	ReplyTo             string         `json:"reply_to"`
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
	if p.ReplyTo != "" && !emailPattern.MatchString(p.ReplyTo) {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid reply_to email: %q", p.ReplyTo)}
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

	fromAddr := map[string]string{"email": params.From}
	if params.FromName != "" {
		fromAddr["name"] = params.FromName
	}

	body := map[string]any{
		"personalizations": []map[string]any{
			{"to": []map[string]string{toAddr}},
		},
		"from": fromAddr,
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

	// POST /mail/send returns 202 Accepted with an empty body on success.
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, "/mail/send", body, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"status": "sent",
		"to":     params.To,
	})
}
