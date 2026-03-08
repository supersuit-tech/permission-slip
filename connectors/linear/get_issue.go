package linear

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getIssueAction implements connectors.Action for linear.get_issue.
type getIssueAction struct {
	conn *LinearConnector
}

type getIssueParams struct {
	IssueID string `json:"issue_id"`
}

func (p *getIssueParams) validate() error {
	if p.IssueID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: issue_id"}
	}
	return nil
}

const getIssueQuery = `query GetIssue($id: String!) {
	issue(id: $id) {
		id
		identifier
		title
		description
		priority
		url
		state {
			id
			name
		}
		assignee {
			id
			name
		}
		team {
			id
			name
		}
		labels {
			nodes {
				id
				name
			}
		}
		cycle {
			id
			name
			number
		}
		createdAt
		updatedAt
	}
}`

type getIssueResponse struct {
	Issue struct {
		ID          string `json:"id"`
		Identifier  string `json:"identifier"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    int    `json:"priority"`
		URL         string `json:"url"`
		State       *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"state"`
		Assignee *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"assignee"`
		Team *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"team"`
		Labels struct {
			Nodes []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"nodes"`
		} `json:"labels"`
		Cycle *struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Number int    `json:"number"`
		} `json:"cycle"`
		CreatedAt string `json:"createdAt"`
		UpdatedAt string `json:"updatedAt"`
	} `json:"issue"`
}

func (a *getIssueAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getIssueParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var resp getIssueResponse
	if err := a.conn.doGraphQL(ctx, req.Credentials, getIssueQuery, map[string]any{"id": params.IssueID}, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp.Issue)
}
