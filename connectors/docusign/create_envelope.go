package docusign

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createEnvelopeAction implements connectors.Action for docusign.create_envelope.
// It creates a draft envelope from a template via POST /envelopes.
type createEnvelopeAction struct {
	conn *DocuSignConnector
}

type createEnvelopeParams struct {
	TemplateID   string              `json:"template_id"`
	EmailSubject string              `json:"email_subject"`
	EmailBody    string              `json:"email_body"`
	Recipients   []envelopeRecipient `json:"recipients"`
}

type envelopeRecipient struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	RoleName string `json:"role_name"`
}

func (p *createEnvelopeParams) validate() error {
	if p.TemplateID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: template_id"}
	}
	if len(p.Recipients) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: recipients (at least one recipient required)"}
	}
	for i, r := range p.Recipients {
		if r.Email == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("recipients[%d].email is required", i)}
		}
		if r.Name == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("recipients[%d].name is required", i)}
		}
		if r.RoleName == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("recipients[%d].role_name is required", i)}
		}
	}
	return nil
}

// createEnvelopeRequest is the DocuSign API request body.
type createEnvelopeRequest struct {
	TemplateID    string                `json:"templateId"`
	EmailSubject  string                `json:"emailSubject,omitempty"`
	EmailBlurb    string                `json:"emailBlurb,omitempty"`
	Status        string                `json:"status"`
	TemplateRoles []templateRoleRequest `json:"templateRoles"`
}

type templateRoleRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	RoleName string `json:"roleName"`
}

type createEnvelopeResponse struct {
	EnvelopeID string `json:"envelopeId"`
	URI        string `json:"uri"`
	StatusDate string `json:"statusDateTime"`
	Status     string `json:"status"`
}

func (a *createEnvelopeAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createEnvelopeParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	accountID, _ := req.Credentials.Get(credKeyAccountID)

	roles := make([]templateRoleRequest, len(params.Recipients))
	for i, r := range params.Recipients {
		roles[i] = templateRoleRequest{
			Email:    r.Email,
			Name:     r.Name,
			RoleName: r.RoleName,
		}
	}

	body := createEnvelopeRequest{
		TemplateID:    params.TemplateID,
		EmailSubject:  params.EmailSubject,
		EmailBlurb:    params.EmailBody,
		Status:        "created", // draft — not sent yet
		TemplateRoles: roles,
	}

	var resp createEnvelopeResponse
	path := accountPath(accountID) + "/envelopes"
	if err := a.conn.doJSON(ctx, "POST", path, req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"envelope_id": resp.EnvelopeID,
		"status":      resp.Status,
		"uri":         resp.URI,
	})
}
