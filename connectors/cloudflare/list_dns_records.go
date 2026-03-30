package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type listDNSRecordsAction struct {
	conn *CloudflareConnector
}

type listDNSRecordsParams struct {
	ZoneID string `json:"zone_id"`
	Type   string `json:"type"`
	Name   string `json:"name"`
	Page   int    `json:"page"`
}

func (p *listDNSRecordsParams) validate() error {
	return requirePathParam("zone_id", p.ZoneID)
}

func (a *listDNSRecordsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listDNSRecordsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	q := url.Values{}
	if params.Type != "" {
		q.Set("type", params.Type)
	}
	if params.Name != "" {
		q.Set("name", params.Name)
	}
	if params.Page > 0 {
		q.Set("page", fmt.Sprintf("%d", params.Page))
	}

	path := fmt.Sprintf("/zones/%s/dns_records", params.ZoneID)
	if qs := q.Encode(); qs != "" {
		path += "?" + qs
	}

	var records []json.RawMessage
	if err := a.conn.doGet(ctx, req.Credentials, path, &records); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{"dns_records": records})
}
