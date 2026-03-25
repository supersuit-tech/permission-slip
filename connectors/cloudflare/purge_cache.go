package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type purgeCacheAction struct {
	conn *CloudflareConnector
}

type purgeCacheParams struct {
	ZoneID   string   `json:"zone_id"`
	PurgeAll bool     `json:"purge_everything"`
	Files    []string `json:"files"`
	Tags     []string `json:"tags"`
	Hosts    []string `json:"hosts"`
}

func (p *purgeCacheParams) validate() error {
	if p.ZoneID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: zone_id"}
	}
	if !p.PurgeAll && len(p.Files) == 0 && len(p.Tags) == 0 && len(p.Hosts) == 0 {
		return &connectors.ValidationError{Message: "must specify purge_everything, files, tags, or hosts"}
	}
	return nil
}

func (a *purgeCacheAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params purgeCacheParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{}
	if params.PurgeAll {
		body["purge_everything"] = true
	}
	if len(params.Files) > 0 {
		body["files"] = params.Files
	}
	if len(params.Tags) > 0 {
		body["tags"] = params.Tags
	}
	if len(params.Hosts) > 0 {
		body["hosts"] = params.Hosts
	}

	var result json.RawMessage
	path := fmt.Sprintf("/zones/%s/purge_cache", params.ZoneID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, path, body, &result); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{"status": "purged", "zone_id": params.ZoneID})
}
