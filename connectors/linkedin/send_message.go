package linkedin

// sendMessageAction implements connectors.Action for linkedin.send_message.
//
// # Access tier requirements
//
// This action requires the "Messaging on behalf of members" developer
// privilege in addition to the w_messages OAuth scope. This privilege is
// only available through LinkedIn's Partner Program or approved messaging
// products. Standard developer app credentials will receive HTTP 403.
//
// LinkedIn API reference:
// https://learn.microsoft.com/en-us/linkedin/shared/integrations/people/messaging-api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type sendMessageAction struct {
	conn *LinkedInConnector
}

type sendMessageParams struct {
	RecipientURN string `json:"recipient_urn"`
	Subject      string `json:"subject"`
	Body         string `json:"body"`
}

const maxMessageBodyLen = 8000
const maxMessageSubjectLen = 200

func (p *sendMessageParams) validate() error {
	if p.RecipientURN == "" {
		return &connectors.ValidationError{Message: "missing required parameter: recipient_urn"}
	}
	if err := validatePersonURN(p.RecipientURN); err != nil {
		return err
	}
	if p.Body == "" {
		return &connectors.ValidationError{Message: "missing required parameter: body"}
	}
	if len(p.Body) > maxMessageBodyLen {
		return &connectors.ValidationError{Message: fmt.Sprintf("body exceeds maximum length of %d characters", maxMessageBodyLen)}
	}
	if len(p.Subject) > maxMessageSubjectLen {
		return &connectors.ValidationError{Message: fmt.Sprintf("subject exceeds maximum length of %d characters", maxMessageSubjectLen)}
	}
	return nil
}

// sendMessageRequest is the LinkedIn REST API request body for POST /rest/messages.
type sendMessageRequest struct {
	Recipients  []string `json:"recipients"`
	Subject     string   `json:"subject,omitempty"`
	Body        string   `json:"body"`
	MessageType string   `json:"messageType"`
}

// Execute sends a direct message to a LinkedIn connection.
func (a *sendMessageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendMessageParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := sendMessageRequest{
		Recipients:  []string{params.RecipientURN},
		Subject:     params.Subject,
		Body:        params.Body,
		MessageType: "MEMBER_TO_MEMBER",
	}

	apiURL := a.conn.restBaseURL + "/messages"
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, apiURL, body, nil, true); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"status":        "sent",
		"recipient_urn": params.RecipientURN,
	})
}
