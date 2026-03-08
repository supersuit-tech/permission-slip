package sendgrid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

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

// suppressionEntry is the raw SendGrid API response shape for an unsubscribe record.
// created is a Unix timestamp; we convert it to ISO-8601 in the action result.
type suppressionEntry struct {
	Created int64  `json:"created"`
	Email   string `json:"email"`
}

// suppressionResult is the human-friendly representation returned to callers.
type suppressionResult struct {
	CreatedAt string `json:"created_at"` // ISO-8601, e.g. "2026-01-14T10:00:00Z"
	Email     string `json:"email"`
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

	var raw []suppressionEntry
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}

	results := make([]suppressionResult, len(raw))
	for i, s := range raw {
		results[i] = suppressionResult{
			CreatedAt: time.Unix(s.Created, 0).UTC().Format(time.RFC3339),
			Email:     s.Email,
		}
	}

	return connectors.JSONResult(map[string]any{
		"suppressions": results,
		"count":        len(results),
	})
}
