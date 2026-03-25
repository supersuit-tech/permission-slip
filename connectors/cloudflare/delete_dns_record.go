package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"

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
	if err := requirePathParam("zone_id", p.ZoneID); err != nil {
		return err
	}
	if err := requirePathParam("record_id", p.RecordID); err != nil {
		return err
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
	if err := a.conn.doDelete(ctx, req.Credentials, path); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{"status": "deleted", "record_id": params.RecordID})
}
