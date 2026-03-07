package docusign

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// sendEnvelopeAction implements connectors.Action for docusign.send_envelope.
// It changes a draft envelope's status to "sent" via PUT /envelopes/{envelopeId}.
type sendEnvelopeAction struct {
	conn *DocuSignConnector
}

type sendEnvelopeParams struct {
	EnvelopeID string `json:"envelope_id"`
}

func (p *sendEnvelopeParams) validate() error {
	if p.EnvelopeID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: envelope_id"}
	}
	return nil
}

type sendEnvelopeRequest struct {
	Status string `json:"status"`
}

type sendEnvelopeResponse struct {
	EnvelopeID string `json:"envelopeId"`
	Status     string `json:"status"`
}

func (a *sendEnvelopeAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendEnvelopeParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	accountID, _ := req.Credentials.Get(credKeyAccountID)

	body := sendEnvelopeRequest{Status: "sent"}

	var resp sendEnvelopeResponse
	path := accountPath(accountID) + "/envelopes/" + params.EnvelopeID
	if err := a.conn.doJSON(ctx, "PUT", path, req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"envelope_id": resp.EnvelopeID,
		"status":      resp.Status,
	})
}
