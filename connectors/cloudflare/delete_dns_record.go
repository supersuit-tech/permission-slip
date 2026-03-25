package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type deleteDNSRecordAction struct {
	conn *CloudflareConnector
}

type deleteDNSRecordParams struct {
	ZoneID   string `json:"zone_id"`
	RecordID string `json:"record_id"`
}

func (p *deleteDNSRecordParams) validate() error {
	if p.ZoneID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: zone_id"}
	}
	if p.RecordID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: record_id"}
	}
	return nil
}

func (a *deleteDNSRecordAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteDNSRecordParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/zones/%s/dns_records/%s", params.ZoneID, params.RecordID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodDelete, path, nil, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{"status": "deleted", "record_id": params.RecordID})
}
