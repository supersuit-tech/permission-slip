package docusign

import (
	"context"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// updateRecipientsAction implements connectors.Action for docusign.update_recipients.
// It updates signers on an envelope via PUT /envelopes/{envelopeId}/recipients.
type updateRecipientsAction struct {
	conn *DocuSignConnector
}

type updateRecipientsParams struct {
	EnvelopeID string        `json:"envelope_id"`
	Signers    []signerParam `json:"signers"`
}

type signerParam struct {
	Email        string `json:"email"`
	Name         string `json:"name"`
	RecipientID  string `json:"recipient_id"`
	RoutingOrder string `json:"routing_order"`
}

func (p *updateRecipientsParams) validate() error {
	if p.EnvelopeID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: envelope_id"}
	}
	if len(p.Signers) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: signers (at least one signer required)"}
	}
	for i, s := range p.Signers {
		if s.Email == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("signers[%d].email is required", i)}
		}
		if s.Name == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("signers[%d].name is required", i)}
		}
		if s.RecipientID == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("signers[%d].recipient_id is required", i)}
		}
		if s.RoutingOrder == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("signers[%d].routing_order is required", i)}
		}
	}
	return nil
}

type updateRecipientsRequest struct {
	Signers []signerRequest `json:"signers"`
}

type signerRequest struct {
	Email        string `json:"email"`
	Name         string `json:"name"`
	RecipientID  string `json:"recipientId"`
	RoutingOrder string `json:"routingOrder"`
}

type updateRecipientsResponse struct {
	Signers []recipientUpdateResult `json:"signers"`
}

type recipientUpdateResult struct {
	RecipientID     string `json:"recipientId"`
	RecipientIDGUID string `json:"recipientIdGuid"`
	ErrorDetails    *struct {
		ErrorCode string `json:"errorCode"`
		Message   string `json:"message"`
	} `json:"errorDetails,omitempty"`
}

func (a *updateRecipientsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateRecipientsParams
	accountID, err := parseParams(req, &params)
	if err != nil {
		return nil, err
	}

	signers := make([]signerRequest, len(params.Signers))
	for i, s := range params.Signers {
		signers[i] = signerRequest{
			Email:        s.Email,
			Name:         s.Name,
			RecipientID:  s.RecipientID,
			RoutingOrder: s.RoutingOrder,
		}
	}

	body := updateRecipientsRequest{Signers: signers}

	var resp updateRecipientsResponse
	path := accountPath(accountID) + "/envelopes/" + url.PathEscape(params.EnvelopeID) + "/recipients"
	if err := a.conn.doJSON(ctx, "PUT", path, req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	type signerResult struct {
		RecipientID string `json:"recipient_id"`
		Error       string `json:"error,omitempty"`
	}

	results := make([]signerResult, len(resp.Signers))
	for i, s := range resp.Signers {
		r := signerResult{RecipientID: s.RecipientID}
		if s.ErrorDetails != nil {
			r.Error = s.ErrorDetails.Message
		}
		results[i] = r
	}

	return connectors.JSONResult(map[string]any{
		"envelope_id": params.EnvelopeID,
		"signers":     results,
	})
}
