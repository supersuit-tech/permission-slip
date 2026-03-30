package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type listZonesAction struct {
	conn *CloudflareConnector
}

type listZonesParams struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Page   int    `json:"page"`
}

func (a *listZonesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listZonesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	q := url.Values{}
	if params.Name != "" {
		q.Set("name", params.Name)
	}
	if params.Status != "" {
		q.Set("status", params.Status)
	}
	if params.Page > 0 {
		q.Set("page", fmt.Sprintf("%d", params.Page))
	}

	path := "/zones"
	if qs := q.Encode(); qs != "" {
		path += "?" + qs
	}

	var zones []json.RawMessage
	if err := a.conn.doGet(ctx, req.Credentials, path, &zones); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{"zones": zones})
}
