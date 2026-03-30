package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
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
	if err := requirePathParam("zone_id", p.ZoneID); err != nil {
		return err
	}
	if err := requirePathParam("record_id", p.RecordID); err != nil {
		return err
	}
	if p.Type == "" && p.Name == "" && p.Content == "" && p.TTL == 0 && p.Proxied == nil {
		return &connectors.ValidationError{Message: "must specify at least one field to update (type, name, content, ttl, or proxied)"}
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
