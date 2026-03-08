package intercom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// sendMessageAction implements connectors.Action for intercom.send_message.
// It sends a proactive outbound message via POST /messages.
// This creates an outbound message to a contact (not a ticket reply).
type sendMessageAction struct {
	conn *IntercomConnector
}

type sendMessageParams struct {
	Body        string `json:"body"`
	MessageType string `json:"message_type"` // "inapp" or "email"
	Subject     string `json:"subject"`      // required for email messages
	FromAdminID string `json:"from_admin_id"`
	ToContactID string `json:"to_contact_id"`
}

var validOutboundMessageTypes = map[string]bool{
	"inapp": true,
	"email": true,
}

func (p *sendMessageParams) validate() error {
	if p.Body == "" {
		return &connectors.ValidationError{Message: "missing required parameter: body"}
	}
	if p.FromAdminID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: from_admin_id"}
	}
	if !isValidIntercomID(p.FromAdminID) {
		return &connectors.ValidationError{Message: "from_admin_id contains invalid characters"}
	}
	if p.ToContactID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: to_contact_id"}
	}
	if !isValidIntercomID(p.ToContactID) {
		return &connectors.ValidationError{Message: "to_contact_id contains invalid characters"}
	}
	msgType := p.MessageType
	if msgType == "" {
		msgType = "inapp"
	}
	if !validOutboundMessageTypes[msgType] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid message_type %q: must be inapp or email", p.MessageType)}
	}
	if msgType == "email" && p.Subject == "" {
		return &connectors.ValidationError{Message: "subject is required for email messages"}
	}
	return nil
}

func (a *sendMessageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendMessageParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	msgType := params.MessageType
	if msgType == "" {
		msgType = "inapp"
	}

	body := map[string]any{
		"body":         params.Body,
		"message_type": msgType,
		"from": map[string]string{
			"type": "admin",
			"id":   params.FromAdminID,
		},
		"to": map[string]string{
			"type": "contact",
			"id":   params.ToContactID,
		},
	}
	if params.Subject != "" {
		body["subject"] = params.Subject
	}

	var resp outboundMessageResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/messages", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
