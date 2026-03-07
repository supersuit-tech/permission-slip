package airtable

import (
	"context"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listBasesAction implements connectors.Action for airtable.list_bases.
type listBasesAction struct {
	conn *AirtableConnector
}

type listBasesParams struct {
	Offset string `json:"offset,omitempty"`
}

func (p *listBasesParams) validate() error { return nil }

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
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	reqURL := a.conn.metaURL + "/bases"
	if params.Offset != "" {
		q := url.Values{}
		q.Set("offset", params.Offset)
		reqURL += "?" + q.Encode()
	}

	var resp listBasesResponse
	if err := a.conn.doRequest(ctx, "GET", reqURL, req.Credentials, nil, &resp); err != nil {
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
