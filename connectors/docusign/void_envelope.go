package docusign

import (
	"context"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// voidEnvelopeAction implements connectors.Action for docusign.void_envelope.
// It voids an in-progress envelope via PUT /envelopes/{envelopeId} with status "voided".
type voidEnvelopeAction struct {
	conn *DocuSignConnector
}

type voidEnvelopeParams struct {
	EnvelopeID string `json:"envelope_id"`
	VoidReason string `json:"void_reason"`
}

func (p *voidEnvelopeParams) validate() error {
	if p.EnvelopeID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: envelope_id"}
	}
	if p.VoidReason == "" {
		return &connectors.ValidationError{Message: "missing required parameter: void_reason"}
	}
	return nil
}

type voidEnvelopeRequest struct {
	Status       string `json:"status"`
	VoidedReason string `json:"voidedReason"`
}

func (a *voidEnvelopeAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params voidEnvelopeParams
	accountID, err := parseParams(req, &params)
	if err != nil {
		return nil, err
	}

	body := voidEnvelopeRequest{
		Status:       "voided",
		VoidedReason: params.VoidReason,
	}

	path := accountPath(accountID) + "/envelopes/" + url.PathEscape(params.EnvelopeID)
	if err := a.conn.doJSON(ctx, "PUT", path, req.Credentials, body, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"envelope_id": params.EnvelopeID,
		"status":      "voided",
	})
}
