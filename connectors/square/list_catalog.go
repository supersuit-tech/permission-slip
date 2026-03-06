package square

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listCatalogAction implements connectors.Action for square.list_catalog.
// It lists catalog objects via GET /v2/catalog/list.
type listCatalogAction struct {
	conn *SquareConnector
}

type listCatalogParams struct {
	Types  string `json:"types,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

func (a *listCatalogAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listCatalogParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	query := url.Values{}
	if params.Types != "" {
		query.Set("types", params.Types)
	}
	if params.Cursor != "" {
		query.Set("cursor", params.Cursor)
	}

	path := "/catalog/list"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var resp struct {
		Objects json.RawMessage `json:"objects"`
		Cursor  string          `json:"cursor,omitempty"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"objects": json.RawMessage(resp.Objects),
	}
	if resp.Cursor != "" {
		result["cursor"] = resp.Cursor
	}

	return connectors.JSONResult(result)
}
