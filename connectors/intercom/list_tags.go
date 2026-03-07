package intercom

import (
	"context"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listTagsAction implements connectors.Action for intercom.list_tags.
// It lists all tags via GET /tags.
type listTagsAction struct {
	conn *IntercomConnector
}

func (a *listTagsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var resp tagsListResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, "/tags", nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
