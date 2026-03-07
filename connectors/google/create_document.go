package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createDocumentAction implements connectors.Action for google.create_document.
// It creates a new Google Doc via the Docs API POST /v1/documents, then
// optionally inserts body text via a batchUpdate.
type createDocumentAction struct {
	conn *GoogleConnector
}

// createDocumentParams is the user-facing parameter schema.
type createDocumentParams struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func (p *createDocumentParams) validate() error {
	if p.Title == "" {
		return &connectors.ValidationError{Message: "missing required parameter: title"}
	}
	return nil
}

// docsCreateRequest is the Google Docs API request body for documents.create.
type docsCreateRequest struct {
	Title string `json:"title"`
}

// docsCreateResponse is the Google Docs API response from documents.create.
type docsCreateResponse struct {
	DocumentID string `json:"documentId"`
	Title      string `json:"title"`
}

// Execute creates a new Google Doc and returns its metadata.
func (a *createDocumentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createDocumentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := docsCreateRequest{Title: params.Title}
	var resp docsCreateResponse

	createURL := a.conn.docsBaseURL + "/v1/documents"
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, createURL, body, &resp); err != nil {
		return nil, err
	}

	documentURL := documentEditURL(resp.DocumentID)

	// If body text was provided, insert it via batchUpdate.
	if params.Body != "" {
		batchReq := docsBatchUpdateRequest{
			Requests: []docsRequest{
				{
					InsertText: &docsInsertTextRequest{
						Text:               params.Body,
						EndOfSegmentLocation: &docsEndOfSegmentLocation{},
					},
				},
			},
		}
		updateURL := a.conn.docsBaseURL + "/v1/documents/" + url.PathEscape(resp.DocumentID) + ":batchUpdate"
		if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, updateURL, batchReq, nil); err != nil {
			// Document was created but body insertion failed. Return the
			// document info so the caller knows what was created, rather
			// than silently orphaning it.
			return connectors.JSONResult(map[string]string{
				"document_id":  resp.DocumentID,
				"title":        resp.Title,
				"document_url": documentURL,
				"warning":      fmt.Sprintf("document created but body insertion failed: %v", err),
			})
		}
	}

	return connectors.JSONResult(map[string]string{
		"document_id":  resp.DocumentID,
		"title":        resp.Title,
		"document_url": documentURL,
	})
}
