package linkedin

// listConnectionsAction implements connectors.Action for linkedin.list_connections.
//
// # Access tier requirements
//
// Listing connections requires the r_network OAuth scope. This scope is
// restricted and requires LinkedIn partner approval. Standard apps may
// not have access to this scope.
//
// LinkedIn API reference:
// https://learn.microsoft.com/en-us/linkedin/shared/integrations/people/connections-api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type listConnectionsAction struct {
	conn *LinkedInConnector
}

type listConnectionsParams struct {
	Count int `json:"count"`
	Start int `json:"start"`
}

const defaultConnectionsCount = 20
const maxConnectionsCount = 500

func (p *listConnectionsParams) validate() error {
	if p.Count < 0 {
		return &connectors.ValidationError{Message: "count must be non-negative"}
	}
	if p.Count > maxConnectionsCount {
		return &connectors.ValidationError{Message: fmt.Sprintf("count must not exceed %d", maxConnectionsCount)}
	}
	if p.Start < 0 {
		return &connectors.ValidationError{Message: "start must be non-negative"}
	}
	return nil
}

// connectionsResponse is the LinkedIn connections list API response.
type connectionsResponse struct {
	Elements []connectionElement `json:"elements"`
	Paging   searchPaging        `json:"paging"`
}

type connectionElement struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Headline  string `json:"headline"`
}

// Execute lists the authenticated user's LinkedIn connections.
func (a *listConnectionsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listConnectionsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	count := params.Count
	if count == 0 {
		count = defaultConnectionsCount
	}

	q := url.Values{}
	q.Set("q", "viewer")
	q.Set("count", strconv.Itoa(count))
	q.Set("start", strconv.Itoa(params.Start))

	apiURL := a.conn.restBaseURL + "/connections?" + q.Encode()

	var resp connectionsResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, apiURL, nil, &resp, true); err != nil {
		return nil, err
	}

	connections := make([]map[string]any, 0, len(resp.Elements))
	for _, el := range resp.Elements {
		connections = append(connections, map[string]any{
			"id":         el.ID,
			"first_name": el.FirstName,
			"last_name":  el.LastName,
			"headline":   el.Headline,
		})
	}

	return connectors.JSONResult(map[string]any{
		"connections": connections,
		"total":       resp.Paging.Total,
		"start":       resp.Paging.Start,
		"count":       resp.Paging.Count,
	})
}
