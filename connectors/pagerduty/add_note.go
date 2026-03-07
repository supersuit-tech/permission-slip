package pagerduty

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// addNoteAction implements connectors.Action for pagerduty.add_note.
type addNoteAction struct {
	conn *PagerDutyConnector
}

type addNoteParams struct {
	IncidentID string `json:"incident_id"`
	Content    string `json:"content"`
}

func (p *addNoteParams) validate() error {
	if p.IncidentID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: incident_id"}
	}
	if p.Content == "" {
		return &connectors.ValidationError{Message: "missing required parameter: content"}
	}
	return nil
}

// Execute adds a note to an existing PagerDuty incident.
func (a *addNoteAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params addNoteParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"note": map[string]any{
			"content": params.Content,
		},
	}

	var respBody json.RawMessage
	path := fmt.Sprintf("/incidents/%s/notes", url.PathEscape(params.IncidentID))
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &respBody); err != nil {
		return nil, err
	}

	return connectors.JSONResult(respBody)
}
