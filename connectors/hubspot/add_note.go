package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// addNoteAction implements connectors.Action for hubspot.add_note.
// It creates a note and associates it with a CRM record via two API calls:
//  1. POST /crm/v3/objects/notes — create the note
//  2. PUT /crm/v3/objects/notes/{note_id}/associations/{object_type}/{object_id}/note_to_{object_type} — associate
type addNoteAction struct {
	conn *HubSpotConnector
}

type addNoteParams struct {
	ObjectType string `json:"object_type"` // contact, deal, or ticket
	ObjectID   string `json:"object_id"`
	Body       string `json:"body"`
}

var validNoteObjectTypes = map[string]bool{
	"contact": true,
	"deal":    true,
	"ticket":  true,
}

func (p *addNoteParams) validate() error {
	if p.ObjectType == "" {
		return &connectors.ValidationError{Message: "missing required parameter: object_type"}
	}
	if !validNoteObjectTypes[p.ObjectType] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid object_type %q: must be contact, deal, or ticket", p.ObjectType)}
	}
	if p.ObjectID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: object_id"}
	}
	if !isValidHubSpotID(p.ObjectID) {
		return &connectors.ValidationError{Message: "object_id must be a numeric HubSpot ID"}
	}
	if p.Body == "" {
		return &connectors.ValidationError{Message: "missing required parameter: body"}
	}
	return nil
}

// pluralObjectType returns the plural form used in the associations API path.
func pluralObjectType(objectType string) string {
	switch objectType {
	case "contact":
		return "contacts"
	case "deal":
		return "deals"
	case "ticket":
		return "tickets"
	default:
		return objectType + "s"
	}
}

func (a *addNoteAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params addNoteParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Step 1: Create the note.
	noteBody := hubspotObjectRequest{
		Properties: map[string]string{
			"hs_note_body": params.Body,
			"hs_timestamp": a.conn.now().UTC().Format("2006-01-02T15:04:05.000Z"),
		},
	}
	var noteResp hubspotObjectResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/crm/v3/objects/notes", noteBody, &noteResp); err != nil {
		return nil, err
	}

	// Step 2: Associate the note with the target object.
	plural := pluralObjectType(params.ObjectType)
	assocPath := fmt.Sprintf("/crm/v3/objects/notes/%s/associations/%s/%s/note_to_%s",
		noteResp.ID, plural, params.ObjectID, params.ObjectType)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, assocPath, nil, nil); err != nil {
		return nil, fmt.Errorf("note created (id=%s) but association failed: %w", noteResp.ID, err)
	}

	return connectors.JSONResult(map[string]string{
		"note_id":     noteResp.ID,
		"object_type": params.ObjectType,
		"object_id":   params.ObjectID,
	})
}
