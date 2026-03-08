package linear

import (
	"context"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listTeamsAction implements connectors.Action for linear.list_teams.
type listTeamsAction struct {
	conn *LinearConnector
}

const listTeamsQuery = `query ListTeams {
	teams {
		nodes {
			id
			name
			key
			description
		}
	}
}`

type listTeamsResponse struct {
	Teams struct {
		Nodes []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Key         string `json:"key"`
			Description string `json:"description"`
		} `json:"nodes"`
	} `json:"teams"`
}

func (a *listTeamsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var resp listTeamsResponse
	if err := a.conn.doGraphQL(ctx, req.Credentials, listTeamsQuery, nil, &resp); err != nil {
		return nil, err
	}

	teams := make([]map[string]string, 0, len(resp.Teams.Nodes))
	for _, t := range resp.Teams.Nodes {
		teams = append(teams, map[string]string{
			"id":          t.ID,
			"name":        t.Name,
			"key":         t.Key,
			"description": t.Description,
		})
	}

	return connectors.JSONResult(map[string]interface{}{
		"teams":       teams,
		"total_count": len(teams),
	})
}
