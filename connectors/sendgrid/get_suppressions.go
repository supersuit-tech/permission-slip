package sendgrid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getSuppressionsAction implements connectors.Action for sendgrid.get_suppressions.
// It retrieves global unsubscribes from the SendGrid v3
// GET /suppression/unsubscribes endpoint.
type getSuppressionsAction struct {
	conn *SendGridConnector
}

type getSuppressionsParams struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func (p *getSuppressionsParams) validate() error {
	if p.Limit < 0 {
		return &connectors.ValidationError{Message: "limit must be non-negative"}
	}
	if p.Offset < 0 {
		return &connectors.ValidationError{Message: "offset must be non-negative"}
	}
	return nil
}

type suppressionEntry struct {
	Created int64  `json:"created"`
	Email   string `json:"email"`
}

// Execute retrieves global unsubscribes via GET /suppression/unsubscribes.
func (a *getSuppressionsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getSuppressionsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := "/suppression/unsubscribes"
	sep := "?"
	if params.Limit > 0 {
		path += sep + "limit=" + strconv.Itoa(params.Limit)
		sep = "&"
	}
	if params.Offset > 0 {
		path += sep + "offset=" + strconv.Itoa(params.Offset)
	}

	var suppressions []suppressionEntry
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, path, nil, &suppressions); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"suppressions": suppressions,
		"count":        len(suppressions),
	})
}
