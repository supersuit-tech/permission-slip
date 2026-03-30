package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type getZoneAction struct {
	conn *CloudflareConnector
}

type getZoneParams struct {
	ZoneID string `json:"zone_id"`
}

func (p *getZoneParams) validate() error {
	return requirePathParam("zone_id", p.ZoneID)
}

func (a *getZoneAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getZoneParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var zone json.RawMessage
	if err := a.conn.doGet(ctx, req.Credentials, "/zones/"+params.ZoneID, &zone); err != nil {
		return nil, err
	}

	return connectors.JSONResult(zone)
}
