package intercom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip/connectors"
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

	q := url.Values{}
	q.Set("per_page", strconv.Itoa(limit))
	if params.State != "" {
		q.Set("state", params.State)
	}
	path := "/conversations?" + q.Encode()

	var resp conversationsResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
