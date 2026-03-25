package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type createDNSRecordAction struct {
	conn *CloudflareConnector
}

type createDNSRecordParams struct {
	ZoneID  string `json:"zone_id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied *bool  `json:"proxied"`
}

func (p *createDNSRecordParams) validate() error {
	if p.ZoneID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: zone_id"}
	}
	if p.Type == "" {
		return &connectors.ValidationError{Message: "missing required parameter: type"}
	}
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	if p.Content == "" {
		return &connectors.ValidationError{Message: "missing required parameter: content"}
	}
	if err := validatePathParam("zone_id", p.ZoneID); err != nil {
		return err
	}
	return nil
}

func (a *createDNSRecordAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createDNSRecordParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"type":    params.Type,
		"name":    params.Name,
		"content": params.Content,
	}
	if params.TTL > 0 {
		body["ttl"] = params.TTL
	}
	if params.Proxied != nil {
		body["proxied"] = *params.Proxied
	}

	var record json.RawMessage
	path := fmt.Sprintf("/zones/%s/dns_records", params.ZoneID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, path, body, &record); err != nil {
		return nil, err
	}

	return connectors.JSONResult(record)
}
