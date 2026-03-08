package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

	path := "/board/" + url.PathEscape(params.BoardID) + "/sprint"
	if params.State != "" {
		path += "?state=" + url.QueryEscape(params.State)
	}

	var resp struct {
		Values []struct {
			ID        int    `json:"id"`
			Name      string `json:"name"`
			State     string `json:"state"`
			Goal      string `json:"goal"`
			StartDate string `json:"startDate"`
			EndDate   string `json:"endDate"`
		} `json:"values"`
	}
	if err := a.conn.doAgile(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	sprints := make([]map[string]interface{}, 0, len(resp.Values))
	for _, s := range resp.Values {
		sprints = append(sprints, map[string]interface{}{
			"id":         s.ID,
			"name":       s.Name,
			"state":      s.State,
			"goal":       s.Goal,
			"start_date": s.StartDate,
			"end_date":   s.EndDate,
		})
	}

	return connectors.JSONResult(map[string]interface{}{
		"sprints":     sprints,
		"total_count": len(sprints),
	})
}
