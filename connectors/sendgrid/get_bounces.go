package sendgrid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getBouncesAction implements connectors.Action for sendgrid.get_bounces.
// It retrieves the bounce list from the SendGrid v3 suppression/bounces endpoint.
type getBouncesAction struct {
	conn *SendGridConnector
}

type getBouncesParams struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Limit     int    `json:"limit"`
	Offset    int    `json:"offset"`
}

func (p *getBouncesParams) validate() error {
	if p.StartTime != "" {
		if _, err := time.Parse(time.RFC3339, p.StartTime); err != nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid start_time format (use RFC3339, e.g. 2026-01-01T00:00:00Z): %v", err)}
		}
	}
	if p.EndTime != "" {
		if _, err := time.Parse(time.RFC3339, p.EndTime); err != nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid end_time format (use RFC3339, e.g. 2026-01-31T23:59:59Z): %v", err)}
		}
	}
	if p.Limit < 0 {
		return &connectors.ValidationError{Message: "limit must be non-negative"}
	}
	if p.Offset < 0 {
		return &connectors.ValidationError{Message: "offset must be non-negative"}
	}
	return nil
}

type bounceEntry struct {
	Created int64  `json:"created"`
	Email   string `json:"email"`
	Reason  string `json:"reason"`
	Status  string `json:"status"`
}

// Execute retrieves bounces via GET /suppression/bounces.
func (a *getBouncesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getBouncesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	q := url.Values{}
	if params.StartTime != "" {
		t, _ := time.Parse(time.RFC3339, params.StartTime)
		q.Set("start_time", strconv.FormatInt(t.Unix(), 10))
	}
	if params.EndTime != "" {
		t, _ := time.Parse(time.RFC3339, params.EndTime)
		q.Set("end_time", strconv.FormatInt(t.Unix(), 10))
	}
	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset > 0 {
		q.Set("offset", strconv.Itoa(params.Offset))
	}

	path := "/suppression/bounces"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}

	var bounces []bounceEntry
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, path, nil, &bounces); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"bounces": bounces,
		"count":   len(bounces),
	})
}
