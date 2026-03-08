package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getSatisfactionRatingsAction implements connectors.Action for zendesk.get_satisfaction_ratings.
// It fetches CSAT (customer satisfaction) ratings via GET /satisfaction_ratings.json.
type getSatisfactionRatingsAction struct {
	conn *ZendeskConnector
}

type getSatisfactionRatingsParams struct {
	Score     string `json:"score"`      // "offered", "unoffered", "good", "bad", "good_with_comment", "bad_with_comment"
	StartTime int64  `json:"start_time"` // Unix timestamp (seconds)
	EndTime   int64  `json:"end_time"`   // Unix timestamp (seconds)
	Limit     int    `json:"limit"`
}

var validSatisfactionScores = map[string]bool{
	"offered":           true,
	"unoffered":         true,
	"good":              true,
	"bad":               true,
	"good_with_comment": true,
	"bad_with_comment":  true,
}

const (
	defaultSatisfactionLimit = 25
	maxSatisfactionLimit     = 100
)

func (p *getSatisfactionRatingsParams) validate() error {
	if p.Score != "" && !validSatisfactionScores[p.Score] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid score %q: must be offered, unoffered, good, bad, good_with_comment, or bad_with_comment", p.Score)}
	}
	return nil
}

type satisfactionRating struct {
	ID        int64  `json:"id"`
	Score     string `json:"score"`
	Comment   string `json:"comment,omitempty"`
	TicketID  int64  `json:"ticket_id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type satisfactionRatingsResponse struct {
	SatisfactionRatings []satisfactionRating `json:"satisfaction_ratings"`
	Count               int                  `json:"count"`
}

func (a *getSatisfactionRatingsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getSatisfactionRatingsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit <= 0 {
		limit = defaultSatisfactionLimit
	}
	if limit > maxSatisfactionLimit {
		limit = maxSatisfactionLimit
	}

	q := url.Values{}
	q.Set("per_page", strconv.Itoa(limit))
	if params.Score != "" {
		q.Set("score", params.Score)
	}
	if params.StartTime > 0 {
		q.Set("start_time", strconv.FormatInt(params.StartTime, 10))
	}
	if params.EndTime > 0 {
		q.Set("end_time", strconv.FormatInt(params.EndTime, 10))
	}
	path := "/satisfaction_ratings.json?" + q.Encode()

	var resp satisfactionRatingsResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
