package sendgrid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listListsAction implements connectors.Action for sendgrid.list_lists.
// It lists all contact lists via GET /marketing/lists, which users need
// to find list_id values for adding subscribers or targeting campaigns.
type listListsAction struct {
	conn *SendGridConnector
}

// Execute lists all contact lists in the SendGrid account.
func (a *listListsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	// Validate parameters (none required, but reject malformed JSON)
	if len(req.Parameters) > 0 {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(req.Parameters, &raw); err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
		}
	}

	var resp struct {
		Result []struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			ContactCount int    `json:"contact_count"`
		} `json:"result"`
	}
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, "/marketing/lists", nil, &resp); err != nil {
		return nil, err
	}

	lists := make([]map[string]any, 0, len(resp.Result))
	for _, l := range resp.Result {
		lists = append(lists, map[string]any{
			"id":            l.ID,
			"name":          l.Name,
			"contact_count": l.ContactCount,
		})
	}

	return connectors.JSONResult(map[string]any{
		"lists": lists,
		"count": len(lists),
	})
}
