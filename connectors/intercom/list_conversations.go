package intercom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listConversationsAction implements connectors.Action for intercom.list_conversations.
// It lists conversations via GET /conversations with optional state filter.
type listConversationsAction struct {
	conn *IntercomConnector
}

type listConversationsParams struct {
	State string `json:"state"` // "open", "closed", "snoozed", or "" for all
	Limit int    `json:"limit"`
}

type intercomConversation struct {
	Type          string `json:"type"`
	ID            string `json:"id"`
	Title         string `json:"title"`
	State         string `json:"state"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
	WaitingSince  int64  `json:"waiting_since"`
	SnoozedUntil  int64  `json:"snoozed_until"`
}

type conversationsResponse struct {
	Type          string                 `json:"type"`
	TotalCount    int                    `json:"total_count"`
	Conversations []intercomConversation `json:"conversations"`
}

var validConversationStates = map[string]bool{
	"open":    true,
	"closed":  true,
	"snoozed": true,
}

const (
	defaultConversationLimit = 20
	maxConversationLimit     = 150
)

func (p *listConversationsParams) validate() error {
	if p.State != "" && !validConversationStates[p.State] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid state %q: must be open, closed, or snoozed", p.State)}
	}
	return nil
}

func (a *listConversationsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listConversationsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit <= 0 {
		limit = defaultConversationLimit
	}
	if limit > maxConversationLimit {
		limit = maxConversationLimit
	}

	path := fmt.Sprintf("/conversations?per_page=%d", limit)
	if params.State != "" {
		path += fmt.Sprintf("&state=%s", params.State)
	}

	var resp conversationsResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
