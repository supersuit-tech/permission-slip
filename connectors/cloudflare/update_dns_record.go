package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type updateDNSRecordAction struct {
	conn *CloudflareConnector
}

type updateDNSRecordParams struct {
	ZoneID   string `json:"zone_id"`
	RecordID string `json:"record_id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Proxied  *bool  `json:"proxied"`
}

func (p *updateDNSRecordParams) validate() error {
	if p.ZoneID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: zone_id"}
	}
	if p.RecordID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: record_id"}
	}
	return nil
}

func (a *updateDNSRecordAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateDNSRecordParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{}
	if params.Type != "" {
		body["type"] = params.Type
	}
	if params.Name != "" {
		body["name"] = params.Name
	}
	if params.Content != "" {
		body["content"] = params.Content
	}
	if params.TTL > 0 {
		body["ttl"] = params.TTL
	}
	if params.Proxied != nil {
		body["proxied"] = *params.Proxied
	}

	var record json.RawMessage
	path := fmt.Sprintf("/zones/%s/dns_records/%s", params.ZoneID, params.RecordID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPatch, path, body, &record); err != nil {
		return nil, err
	}

	return connectors.JSONResult(record)
}
