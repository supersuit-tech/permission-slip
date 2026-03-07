package walmart

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getTrendingAction implements connectors.Action for walmart.get_trending.
// It retrieves trending products via GET /trends.
type getTrendingAction struct {
	conn *WalmartConnector
}

func (a *getTrendingAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, "/trends?format=json", &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(json.RawMessage(resp))
}
