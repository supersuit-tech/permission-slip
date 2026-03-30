package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// deleteContactAction implements connectors.Action for hubspot.delete_contact.
// It archives (soft-deletes) a contact via DELETE /crm/v3/objects/contacts/{contact_id}.
// HubSpot's "delete" is actually an archive — the record can be restored.
type deleteContactAction struct {
	conn *HubSpotConnector
}

type deleteContactParams struct {
	ContactID string `json:"contact_id"`
}

func (p *deleteContactParams) validate() error {
	if p.ContactID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: contact_id"}
	}
	if !isValidHubSpotID(p.ContactID) {
		return &connectors.ValidationError{Message: "contact_id must be a numeric HubSpot ID"}
	}
	return nil
}

func (a *deleteContactAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteContactParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/crm/v3/objects/contacts/%s", params.ContactID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodDelete, path, nil, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"contact_id": params.ContactID,
		"archived":   true,
	})
}
