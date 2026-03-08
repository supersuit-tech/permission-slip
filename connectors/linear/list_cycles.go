package linear

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listCyclesAction implements connectors.Action for linear.list_cycles.
type listCyclesAction struct {
	conn *LinearConnector
}

type listCyclesParams struct {
	TeamID string `json:"team_id"`
}

func (p *listCyclesParams) validate() error {
	if p.TeamID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: team_id"}
	}
	return nil
}

const listCyclesQuery = `query ListCycles($teamId: String) {
	cycles(filter: { team: { id: { eq: $teamId } } }) {
		nodes {
			id
			name
			number
			startsAt
			endsAt
			completedAt
		}
	}
}`

type listCyclesResponse struct {
	Cycles struct {
		Nodes []struct {
			ID          string  `json:"id"`
			Name        string  `json:"name"`
			Number      int     `json:"number"`
			StartsAt    string  `json:"startsAt"`
			EndsAt      string  `json:"endsAt"`
			CompletedAt *string `json:"completedAt"`
		} `json:"nodes"`
	} `json:"cycles"`
}

func (a *listCyclesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listCyclesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	vars := map[string]any{"teamId": params.TeamID}
	var resp listCyclesResponse
	if err := a.conn.doGraphQL(ctx, req.Credentials, listCyclesQuery, vars, &resp); err != nil {
		return nil, err
	}

	cycles := make([]map[string]interface{}, 0, len(resp.Cycles.Nodes))
	for _, c := range resp.Cycles.Nodes {
		cycle := map[string]interface{}{
			"id":        c.ID,
			"name":      c.Name,
			"number":    c.Number,
			"starts_at": c.StartsAt,
			"ends_at":   c.EndsAt,
		}
		if c.CompletedAt != nil {
			cycle["completed_at"] = *c.CompletedAt
		}
		cycles = append(cycles, cycle)
	}

	return connectors.JSONResult(map[string]interface{}{
		"cycles":      cycles,
		"total_count": len(cycles),
	})
}
