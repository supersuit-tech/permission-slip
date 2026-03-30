package docusign

import (
	"context"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// checkStatusAction implements connectors.Action for docusign.check_status.
// It retrieves envelope details via GET /envelopes/{envelopeId} and optionally
// fetches per-recipient signing status.
type checkStatusAction struct {
	conn *DocuSignConnector
}

type checkStatusParams struct {
	EnvelopeID        string `json:"envelope_id"`
	IncludeRecipients *bool  `json:"include_recipients,omitempty"`
}

func (p *checkStatusParams) validate() error {
	if isBlank(p.EnvelopeID) {
		return &connectors.ValidationError{Message: "missing required parameter: envelope_id"}
	}
	return nil
}

// includeRecipients returns whether to fetch recipient details (defaults to true).
func (p *checkStatusParams) includeRecipients() bool {
	if p.IncludeRecipients == nil {
		return true
	}
	return *p.IncludeRecipients
}

type checkStatusResponse struct {
	EnvelopeID    string `json:"envelopeId"`
	Status        string `json:"status"`
	StatusDate    string `json:"statusChangedDateTime"`
	EmailSubject  string `json:"emailSubject"`
	SentDate      string `json:"sentDateTime"`
	DeliveredDate string `json:"deliveredDateTime"`
	CompletedDate string `json:"completedDateTime"`
	VoidedDate    string `json:"voidedDateTime"`
	VoidedReason  string `json:"voidedReason"`
	DeclinedDate  string `json:"declinedDateTime"`
	RecipientsURI string `json:"recipientsUri"`
	DocumentsURI  string `json:"documentsUri"`
}

type recipientsResponse struct {
	Signers []recipientDetail `json:"signers"`
}

type recipientDetail struct {
	RecipientID   string `json:"recipientId"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	Status        string `json:"status"`
	RoutingOrder  string `json:"routingOrder"`
	SignedDate    string `json:"signedDateTime"`
	DeliveredDate string `json:"deliveredDateTime"`
	DeclinedDate  string `json:"declinedDateTime"`
	DeclineReason string `json:"declinedReason"`
}

func (a *checkStatusAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params checkStatusParams
	accountID, err := parseParams(req, &params)
	if err != nil {
		return nil, err
	}

	var resp checkStatusResponse
	path := accountPath(accountID) + "/envelopes/" + url.PathEscape(params.EnvelopeID)
	if err := a.conn.doJSON(ctx, "GET", path, req.Credentials, nil, &resp); err != nil {
		return nil, err
	}

	result := map[string]any{
		"envelope_id":   resp.EnvelopeID,
		"status":        resp.Status,
		"status_date":   resp.StatusDate,
		"email_subject": resp.EmailSubject,
	}
	if resp.SentDate != "" {
		result["sent_date"] = resp.SentDate
	}
	if resp.DeliveredDate != "" {
		result["delivered_date"] = resp.DeliveredDate
	}
	if resp.CompletedDate != "" {
		result["completed_date"] = resp.CompletedDate
	}
	if resp.VoidedDate != "" {
		result["voided_date"] = resp.VoidedDate
		result["voided_reason"] = resp.VoidedReason
	}
	if resp.DeclinedDate != "" {
		result["declined_date"] = resp.DeclinedDate
	}

	// Fetch per-recipient signing status when requested (default: true).
	if params.includeRecipients() {
		var recipResp recipientsResponse
		recipPath := accountPath(accountID) + "/envelopes/" + url.PathEscape(params.EnvelopeID) + "/recipients"
		if err := a.conn.doJSON(ctx, "GET", recipPath, req.Credentials, nil, &recipResp); err == nil {
			type recipientResult struct {
				RecipientID   string `json:"recipient_id"`
				Name          string `json:"name"`
				Email         string `json:"email"`
				Status        string `json:"status"`
				RoutingOrder  string `json:"routing_order,omitempty"`
				SignedDate    string `json:"signed_date,omitempty"`
				DeclinedDate  string `json:"declined_date,omitempty"`
				DeclineReason string `json:"decline_reason,omitempty"`
			}
			recipients := make([]recipientResult, len(recipResp.Signers))
			for i, s := range recipResp.Signers {
				recipients[i] = recipientResult{
					RecipientID:   s.RecipientID,
					Name:          s.Name,
					Email:         s.Email,
					Status:        s.Status,
					RoutingOrder:  s.RoutingOrder,
					SignedDate:    s.SignedDate,
					DeclinedDate:  s.DeclinedDate,
					DeclineReason: s.DeclineReason,
				}
			}
			result["recipients"] = recipients
			result["recipients_included"] = true
		} else {
			// Recipient fetch failed — still return envelope status, but flag
			// that recipients are missing so the caller knows the result is partial.
			result["recipients_included"] = false
		}
	}

	return connectors.JSONResult(result)
}
