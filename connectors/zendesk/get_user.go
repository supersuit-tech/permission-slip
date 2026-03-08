package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getUserAction implements connectors.Action for zendesk.get_user.
// It fetches a user by ID via GET /users/{id}.json.
type getUserAction struct {
	conn *ZendeskConnector
}

type getUserParams struct {
	UserID int64 `json:"user_id"`
}

func (p *getUserParams) validate() error {
	if !isValidZendeskID(p.UserID) {
		return &connectors.ValidationError{Message: "missing or invalid required parameter: user_id"}
	}
	return nil
}

func (a *getUserAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getUserParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/users/%d.json", params.UserID)
	var resp userResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
