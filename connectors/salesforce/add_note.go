package salesforce

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// addNoteAction implements connectors.Action for salesforce.add_note.
// It creates a ContentNote and links it to a parent record via ContentDocumentLink.
type addNoteAction struct {
	conn *SalesforceConnector
}

type addNoteParams struct {
	ParentID string `json:"parent_id"`
	Title    string `json:"title"`
	Body     string `json:"body"`
}

func (p *addNoteParams) validate() error {
	if p.ParentID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: parent_id"}
	}
	if err := validateRecordID(p.ParentID, "parent_id"); err != nil {
		return err
	}
	if p.Title == "" {
		return &connectors.ValidationError{Message: "missing required parameter: title"}
	}
	return nil
}

// sfContentNoteResponse is the Salesforce response from creating a ContentNote.
type sfContentNoteResponse struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
}

func (a *addNoteAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params addNoteParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	baseURL, err := a.conn.apiBaseURL(req.Credentials)
	if err != nil {
		return nil, err
	}

	// Step 1: Create ContentNote.
	// ContentNote.Content must be base64-encoded.
	noteBody := params.Body
	if noteBody == "" {
		noteBody = " " // Salesforce requires non-empty Content
	}
	noteFields := map[string]string{
		"Title":   params.Title,
		"Content": base64.StdEncoding.EncodeToString([]byte(noteBody)),
	}

	noteURL := baseURL + "/sobjects/ContentNote/"
	var noteResp sfContentNoteResponse
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, noteURL, noteFields, &noteResp); err != nil {
		return nil, err
	}

	// Step 2: Link the ContentNote to the parent record via ContentDocumentLink.
	// The ContentNote ID is the ContentDocument ID for linking purposes.
	linkFields := map[string]string{
		"ContentDocumentId": noteResp.ID,
		"LinkedEntityId":    params.ParentID,
		"ShareType":         "V", // Viewer
	}

	linkURL := baseURL + "/sobjects/ContentDocumentLink/"
	var linkResp sfCreateResponse
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, linkURL, linkFields, &linkResp); err != nil {
		// Note was created but linking failed — return partial success.
		return connectors.JSONResult(map[string]any{
			"note_id": noteResp.ID,
			"success": false,
			"warning": fmt.Sprintf("note created but linking to parent failed: %v", err),
		})
	}

	return connectors.JSONResult(map[string]any{
		"note_id": noteResp.ID,
		"link_id": linkResp.ID,
		"success": true,
	})
}
