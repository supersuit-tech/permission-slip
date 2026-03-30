package sendgrid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
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

	// parsedStart/parsedEnd hold the validated time.Time values so Execute
	// doesn't need to re-parse (and cannot silently discard parse errors).
	parsedStart time.Time
	parsedEnd   time.Time
}

func (p *getBouncesParams) validate() error {
	if p.StartTime != "" {
		t, err := time.Parse(time.RFC3339, p.StartTime)
		if err != nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid start_time format (use RFC3339, e.g. 2026-01-01T00:00:00Z): %v", err)}
		}
		p.parsedStart = t
	}
	if p.EndTime != "" {
		t, err := time.Parse(time.RFC3339, p.EndTime)
		if err != nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid end_time format (use RFC3339, e.g. 2026-01-31T23:59:59Z): %v", err)}
		}
		p.parsedEnd = t
	}
	if p.Limit < 0 {
		return &connectors.ValidationError{Message: "limit must be non-negative"}
	}
	if p.Offset < 0 {
		return &connectors.ValidationError{Message: "offset must be non-negative"}
	}
	return nil
}

// bounceEntry is the raw SendGrid API response shape for a bounce record.
// created is a Unix timestamp; we convert it to ISO-8601 in the action result.
type bounceEntry struct {
	Created int64  `json:"created"`
	Email   string `json:"email"`
	Reason  string `json:"reason"`
	Status  string `json:"status"`
}

// bounceResult is the human-friendly representation returned to callers.
type bounceResult struct {
	CreatedAt string `json:"created_at"` // ISO-8601, e.g. "2026-01-14T10:00:00Z"
	Email     string `json:"email"`
	Reason    string `json:"reason"`
	Status    string `json:"status"`
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
	if !params.parsedStart.IsZero() {
		q.Set("start_time", strconv.FormatInt(params.parsedStart.Unix(), 10))
	}
	if !params.parsedEnd.IsZero() {
		q.Set("end_time", strconv.FormatInt(params.parsedEnd.Unix(), 10))
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

	var raw []bounceEntry
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}

	results := make([]bounceResult, len(raw))
	for i, b := range raw {
		results[i] = bounceResult{
			CreatedAt: unixToISO(b.Created),
			Email:     b.Email,
			Reason:    b.Reason,
			Status:    b.Status,
		}
	}

	return connectors.JSONResult(map[string]any{
		"bounces": results,
		"count":   len(results),
	})
}
