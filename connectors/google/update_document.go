package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// updateDocumentAction implements connectors.Action for google.update_document.
// It inserts text into an existing Google Doc via the Docs API
// POST /v1/documents/{documentId}:batchUpdate.
type updateDocumentAction struct {
	conn *GoogleConnector
}

// updateDocumentParams is the user-facing parameter schema.
type updateDocumentParams struct {
	DocumentID string `json:"document_id"`
	Text       string `json:"text"`
	Index      int    `json:"index"`
}

func (p *updateDocumentParams) validate() error {
	if p.DocumentID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: document_id"}
	}
	if p.Text == "" {
		return &connectors.ValidationError{Message: "missing required parameter: text"}
	}
	if p.Index < 0 {
		return &connectors.ValidationError{Message: "index must be non-negative"}
	}
	return nil
}

// Shared Docs API batchUpdate request types, used by both create_document
// (for inserting initial body) and update_document.

type docsBatchUpdateRequest struct {
	Requests []docsRequest `json:"requests"`
}

type docsRequest struct {
	InsertText *docsInsertTextRequest `json:"insertText,omitempty"`
}

type docsInsertTextRequest struct {
	Text                 string                      `json:"text"`
	Location             *docsLocation               `json:"location,omitempty"`
	EndOfSegmentLocation *docsEndOfSegmentLocation   `json:"endOfSegmentLocation,omitempty"`
}

type docsLocation struct {
	Index int `json:"index"`
}

type docsEndOfSegmentLocation struct{}

// Execute inserts text into a Google Doc and returns success status.
func (a *updateDocumentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateDocumentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var insertReq docsRequest
	if params.Index > 0 {
		insertReq.InsertText = &docsInsertTextRequest{
			Text:     params.Text,
			Location: &docsLocation{Index: params.Index},
		}
	} else {
		insertReq.InsertText = &docsInsertTextRequest{
			Text:                 params.Text,
			EndOfSegmentLocation: &docsEndOfSegmentLocation{},
		}
	}

	batchReq := docsBatchUpdateRequest{
		Requests: []docsRequest{insertReq},
	}

	updateURL := a.conn.docsBaseURL + "/v1/documents/" + url.PathEscape(params.DocumentID) + ":batchUpdate"
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, updateURL, batchReq, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"document_id": params.DocumentID,
		"status":      "updated",
	})
}
