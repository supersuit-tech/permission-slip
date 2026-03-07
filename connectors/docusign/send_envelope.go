package docusign

import (
	"context"
	"net/url"

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
	accountID, err := parseParams(req, &params)
	if err != nil {
		return nil, err
	}

	body := sendEnvelopeRequest{Status: "sent"}

	var resp sendEnvelopeResponse
	path := accountPath(accountID) + "/envelopes/" + url.PathEscape(params.EnvelopeID)
	if err := a.conn.doJSON(ctx, "PUT", path, req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"envelope_id": resp.EnvelopeID,
		"status":      resp.Status,
	})
}
