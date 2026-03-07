package docusign

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// downloadSignedAction implements connectors.Action for docusign.download_signed.
// It downloads the signed document PDF via GET /envelopes/{envelopeId}/documents/{documentId}.
type downloadSignedAction struct {
	conn *DocuSignConnector
}

type downloadSignedParams struct {
	EnvelopeID string `json:"envelope_id"`
	DocumentID string `json:"document_id"`
}

func (p *downloadSignedParams) validate() error {
	if p.EnvelopeID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: envelope_id"}
	}
	return nil
}

func (a *downloadSignedAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params downloadSignedParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	accountID, _ := req.Credentials.Get(credKeyAccountID)

	documentID := params.DocumentID
	if documentID == "" {
		documentID = "combined"
	}

	path := accountPath(accountID) + "/envelopes/" + params.EnvelopeID + "/documents/" + documentID
	body, err := a.conn.doRaw(ctx, "GET", path, req.Credentials)
	if err != nil {
		return nil, err
	}

	encoded := base64.StdEncoding.EncodeToString(body)

	return connectors.JSONResult(map[string]string{
		"envelope_id":     params.EnvelopeID,
		"document_id":     documentID,
		"content":         encoded,
		"encoding":        "base64",
		"mime_type":       "application/pdf",
		"file_size_bytes": strconv.Itoa(len(body)),
	})
}
