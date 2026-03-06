package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// sendEmailAction implements connectors.Action for microsoft.send_email.
// It sends an email via POST /me/sendMail using the Microsoft Graph API.
type sendEmailAction struct {
	conn *MicrosoftConnector
}

// sendEmailParams is the user-facing parameter schema.
type sendEmailParams struct {
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	CC      []string `json:"cc,omitempty"`
}

func (p *sendEmailParams) validate() error {
	if len(p.To) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: to"}
	}
	for i, addr := range p.To {
		if err := validateEmail("to", i, addr); err != nil {
			return err
		}
	}
	for i, addr := range p.CC {
		if err := validateEmail("cc", i, addr); err != nil {
			return err
		}
	}
	if p.Subject == "" {
		return &connectors.ValidationError{Message: "missing required parameter: subject"}
	}
	if p.Body == "" {
		return &connectors.ValidationError{Message: "missing required parameter: body"}
	}
	return nil
}

// graphRecipient represents a recipient in the Microsoft Graph email format.
type graphRecipient struct {
	EmailAddress struct {
		Address string `json:"address"`
	} `json:"emailAddress"`
}

// graphSendMailRequest is the Microsoft Graph API request body for /me/sendMail.
type graphSendMailRequest struct {
	Message struct {
		Subject      string           `json:"subject"`
		Body         graphEmailBody   `json:"body"`
		ToRecipients []graphRecipient `json:"toRecipients"`
		CCRecipients []graphRecipient `json:"ccRecipients,omitempty"`
	} `json:"message"`
}

type graphEmailBody struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

// validateEmail performs basic email address validation for a recipient list entry.
func validateEmail(field string, idx int, addr string) error {
	if addr == "" {
		return &connectors.ValidationError{Message: fmt.Sprintf("%s[%d] is empty", field, idx)}
	}
	if !strings.Contains(addr, "@") || strings.HasPrefix(addr, "@") || strings.HasSuffix(addr, "@") {
		return &connectors.ValidationError{Message: fmt.Sprintf("%s[%d] is not a valid email address: %q", field, idx, addr)}
	}
	return nil
}

// detectContentType returns "HTML" if the body contains HTML tags, otherwise "Text".
func detectContentType(body string) string {
	if strings.Contains(body, "<") && strings.Contains(body, ">") {
		return "HTML"
	}
	return "Text"
}

func toGraphRecipients(addrs []string) []graphRecipient {
	recipients := make([]graphRecipient, len(addrs))
	for i, addr := range addrs {
		recipients[i].EmailAddress.Address = addr
	}
	return recipients
}

// Execute sends an email via the Microsoft Graph API.
// The sendMail endpoint returns 202 Accepted with no body on success.
func (a *sendEmailAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendEmailParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var graphReq graphSendMailRequest
	graphReq.Message.Subject = params.Subject
	graphReq.Message.Body = graphEmailBody{
		ContentType: detectContentType(params.Body),
		Content:     params.Body,
	}
	graphReq.Message.ToRecipients = toGraphRecipients(params.To)
	if len(params.CC) > 0 {
		graphReq.Message.CCRecipients = toGraphRecipients(params.CC)
	}

	if err := a.conn.doRequest(ctx, http.MethodPost, "/me/sendMail", req.Credentials, graphReq, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"status": "sent",
	})
}
