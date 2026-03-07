package docusign

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// checkStatusAction implements connectors.Action for docusign.check_status.
// It retrieves envelope details via GET /envelopes/{envelopeId}.
type checkStatusAction struct {
	conn *DocuSignConnector
}

type checkStatusParams struct {
	EnvelopeID string `json:"envelope_id"`
}

func (p *checkStatusParams) validate() error {
	if p.EnvelopeID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: envelope_id"}
	}
	return nil
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

func (a *checkStatusAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params checkStatusParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	accountID, _ := req.Credentials.Get(credKeyAccountID)

	var resp checkStatusResponse
	path := accountPath(accountID) + "/envelopes/" + params.EnvelopeID
	if err := a.conn.doJSON(ctx, "GET", path, req.Credentials, nil, &resp); err != nil {
		return nil, err
	}

	result := map[string]string{
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

	return connectors.JSONResult(result)
}
