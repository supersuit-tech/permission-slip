package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// sendEmailReplyAction implements connectors.Action for microsoft.send_email_reply.
// It replies to an existing message via POST /me/messages/{id}/reply.
type sendEmailReplyAction struct {
	conn *MicrosoftConnector
}

type sendEmailReplyParams struct {
	MessageID string `json:"message_id"`
	Body      string `json:"body"`
	HTML      *bool  `json:"html,omitempty"`
}

func (p *sendEmailReplyParams) validate() error {
	if err := validateGraphID("message_id", p.MessageID); err != nil {
		return err
	}
	if p.Body == "" {
		return &connectors.ValidationError{Message: "missing required parameter: body"}
	}
	return nil
}

// ValidateRequest validates parameters at approval request time.
func (a *sendEmailReplyAction) ValidateRequest(params json.RawMessage) error {
	var p sendEmailReplyParams
	if err := json.Unmarshal(params, &p); err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	return p.validate()
}

type graphReplyRequest struct {
	Comment string `json:"comment,omitempty"`
	Message struct {
		Body graphEmailBody `json:"body"`
	} `json:"message,omitempty"`
}

// Execute sends a reply using the Graph reply endpoint.
func (a *sendEmailReplyAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendEmailReplyParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := "/me/messages/" + url.PathEscape(params.MessageID) + "/reply"
	ct := graphBodyContentType(params.HTML)
	var body graphReplyRequest
	if ct == "Text" {
		body.Comment = params.Body
	} else {
		body.Message.Body = graphEmailBody{ContentType: "HTML", Content: params.Body}
	}

	if err := a.conn.doRequest(ctx, http.MethodPost, path, req.Credentials, body, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{"status": "sent"})
}
