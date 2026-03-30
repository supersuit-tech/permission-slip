package linear

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listLabelsAction implements connectors.Action for linear.list_labels.
type listLabelsAction struct {
	conn *LinearConnector
}

type listLabelsParams struct {
	TeamID string `json:"team_id"`
}

const listLabelsQuery = `query ListLabels {
	issueLabels {
		nodes {
			id
			name
			color
			isGroup
		}
	}
}`

const listLabelsForTeamQuery = `query ListLabelsForTeam($teamId: String) {
	issueLabels(filter: { team: { id: { eq: $teamId } } }) {
		nodes {
			id
			name
			color
			isGroup
		}
	}
}`

type listLabelsResponse struct {
	IssueLabels struct {
		Nodes []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Color   string `json:"color"`
			IsGroup bool   `json:"isGroup"`
		} `json:"nodes"`
	} `json:"issueLabels"`
}

func (a *listLabelsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listLabelsParams
	if req.Parameters != nil {
		if err := json.Unmarshal(req.Parameters, &params); err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
		}
	}

	var resp listLabelsResponse
	if params.TeamID != "" {
		vars := map[string]any{"teamId": params.TeamID}
		if err := a.conn.doGraphQL(ctx, req.Credentials, listLabelsForTeamQuery, vars, &resp); err != nil {
			return nil, err
		}
	} else {
		if err := a.conn.doGraphQL(ctx, req.Credentials, listLabelsQuery, nil, &resp); err != nil {
			return nil, err
		}
	}

	labels := make([]map[string]interface{}, 0, len(resp.IssueLabels.Nodes))
	for _, l := range resp.IssueLabels.Nodes {
		labels = append(labels, map[string]interface{}{
			"id":       l.ID,
			"name":     l.Name,
			"color":    l.Color,
			"is_group": l.IsGroup,
		})
	}

	return connectors.JSONResult(map[string]interface{}{
		"labels":      labels,
		"total_count": len(labels),
	})
}
