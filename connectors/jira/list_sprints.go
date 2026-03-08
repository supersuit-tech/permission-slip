package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listSprintsAction implements connectors.Action for jira.list_sprints.
// It lists sprints in a board via GET /rest/agile/1.0/board/{boardId}/sprint.
type listSprintsAction struct {
	conn *JiraConnector
}

type listSprintsParams struct {
	BoardID string `json:"board_id"`
	State   string `json:"state"`
}

func (p *listSprintsParams) validate() error {
	p.BoardID = strings.TrimSpace(p.BoardID)
	if p.BoardID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: board_id"}
	}
	p.State = strings.TrimSpace(p.State)
	if p.State != "" {
		valid := map[string]bool{"future": true, "active": true, "closed": true}
		if !valid[p.State] {
			return &connectors.ValidationError{Message: "state must be one of: future, active, closed"}
		}
	}
	return nil
}

func (a *listSprintsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listSprintsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := "/board/" + params.BoardID + "/sprint"
	if params.State != "" {
		path += "?state=" + params.State
	}

	var resp json.RawMessage
	if err := a.conn.doAgile(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
