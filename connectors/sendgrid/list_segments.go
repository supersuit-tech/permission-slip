package sendgrid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listSegmentsAction implements connectors.Action for sendgrid.list_segments.
// It lists all contact segments via GET /marketing/segments/2.0.
type listSegmentsAction struct {
	conn *SendGridConnector
}

// Execute lists all contact segments in the SendGrid account.
func (a *listSegmentsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	// Validate parameters (none required, but reject malformed JSON)
	if len(req.Parameters) > 0 {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(req.Parameters, &raw); err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
		}
	}

	var resp struct {
		Results []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			ContactsCount int    `json:"contacts_count"`
			CreatedAt     string `json:"created_at"`
			UpdatedAt     string `json:"updated_at"`
		} `json:"results"`
	}
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, "/marketing/segments/2.0", nil, &resp); err != nil {
		return nil, err
	}

	segments := make([]map[string]any, 0, len(resp.Results))
	for _, s := range resp.Results {
		segments = append(segments, map[string]any{
			"id":             s.ID,
			"name":           s.Name,
			"contacts_count": s.ContactsCount,
			"created_at":     s.CreatedAt,
			"updated_at":     s.UpdatedAt,
		})
	}

	return connectors.JSONResult(map[string]any{
		"segments": segments,
		"count":    len(segments),
	})
}
