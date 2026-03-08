package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createUserAction implements connectors.Action for zendesk.create_user.
// It creates a new end-user or agent via POST /users.json.
type createUserAction struct {
	conn *ZendeskConnector
}

type createUserParams struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Role     string `json:"role"` // "end-user", "agent", or "admin"
	Verified bool   `json:"verified"`
}

var validZendeskUserRoles = map[string]bool{
	"end-user": true,
	"agent":    true,
	"admin":    true,
}

func (p *createUserParams) validate() error {
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	if p.Role != "" && !validZendeskUserRoles[p.Role] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid role %q: must be end-user, agent, or admin", p.Role)}
	}
	return nil
}

type zendeskUser struct {
	ID       int64  `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Role     string `json:"role,omitempty"`
	Verified bool   `json:"verified,omitempty"`
}

type userResponse struct {
	User zendeskUser `json:"user"`
}

func (a *createUserAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createUserParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	u := zendeskUser{
		Name:     params.Name,
		Verified: params.Verified,
	}
	if params.Email != "" {
		u.Email = params.Email
	}
	if params.Phone != "" {
		u.Phone = params.Phone
	}
	if params.Role != "" {
		u.Role = params.Role
	}

	body := map[string]zendeskUser{"user": u}
	var resp userResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/users.json", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
