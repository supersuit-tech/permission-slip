package airtable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listBasesAction implements connectors.Action for airtable.list_bases.
type listBasesAction struct {
	conn *AirtableConnector
}

type listBasesParams struct {
	Offset string `json:"offset,omitempty"`
}

type listBasesResponse struct {
	Bases  []baseEntry `json:"bases"`
	Offset string      `json:"offset,omitempty"`
}

type baseEntry struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	PermissionLevel string `json:"permissionLevel"`
}

type listBasesResult struct {
	Bases  []baseSummary `json:"bases"`
	Offset string        `json:"offset,omitempty"`
}

type baseSummary struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	PermissionLevel string `json:"permission_level"`
}

// Execute lists all bases accessible to the authenticated user.
func (a *listBasesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listBasesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	url := a.conn.metaURL + "/bases"
	if params.Offset != "" {
		url += "?offset=" + params.Offset
	}

	var resp listBasesResponse
	if err := a.conn.doRequest(ctx, "GET", url, req.Credentials, nil, &resp); err != nil {
		return nil, err
	}

	result := listBasesResult{
		Bases:  make([]baseSummary, 0, len(resp.Bases)),
		Offset: resp.Offset,
	}
	for _, b := range resp.Bases {
		result.Bases = append(result.Bases, baseSummary{
			ID:              b.ID,
			Name:            b.Name,
			PermissionLevel: b.PermissionLevel,
		})
	}

	return connectors.JSONResult(result)
}
