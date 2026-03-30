package sendgrid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listSendersAction implements connectors.Action for sendgrid.list_senders.
// It lists verified sender identities via GET /verified_senders, which users
// need to find their sender_id for creating campaigns.
type listSendersAction struct {
	conn *SendGridConnector
}

// Execute lists all verified sender identities in the SendGrid account.
func (a *listSendersAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	// Validate parameters (none required, but reject malformed JSON)
	if len(req.Parameters) > 0 {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(req.Parameters, &raw); err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
		}
	}

	var resp struct {
		Results []struct {
			ID        int    `json:"id"`
			Nickname  string `json:"nickname"`
			FromEmail string `json:"from_email"`
			FromName  string `json:"from_name"`
			ReplyTo   string `json:"reply_to"`
			Verified  bool   `json:"verified"`
		} `json:"results"`
	}
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, "/verified_senders", nil, &resp); err != nil {
		return nil, err
	}

	senders := make([]map[string]any, 0, len(resp.Results))
	for _, s := range resp.Results {
		senders = append(senders, map[string]any{
			"id":         s.ID,
			"nickname":   s.Nickname,
			"from_email": s.FromEmail,
			"from_name":  s.FromName,
			"reply_to":   s.ReplyTo,
			"verified":   s.Verified,
		})
	}

	return connectors.JSONResult(map[string]any{
		"senders": senders,
		"count":   len(senders),
	})
}
