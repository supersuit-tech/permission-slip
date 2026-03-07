package zoom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listMeetingsAction implements connectors.Action for zoom.list_meetings.
// It lists meetings via GET /users/me/meetings.
type listMeetingsAction struct {
	conn *ZoomConnector
}

type listMeetingsParams struct {
	Type     string `json:"type"`
	PageSize int    `json:"page_size"`
}

func (p *listMeetingsParams) normalize() {
	if p.Type == "" {
		p.Type = "upcoming"
	}
	if p.PageSize <= 0 {
		p.PageSize = 30
	}
	if p.PageSize > 300 {
		p.PageSize = 300
	}
}

var validMeetingListTypes = map[string]bool{
	"scheduled": true,
	"live":      true,
	"upcoming":  true,
}

func (p *listMeetingsParams) validate() error {
	if p.Type != "" && !validMeetingListTypes[p.Type] {
		return &connectors.ValidationError{Message: fmt.Sprintf("type must be one of: scheduled, live, upcoming; got %q", p.Type)}
	}
	return nil
}

// zoomMeetingsResponse is the Zoom API response from GET /users/me/meetings.
type zoomMeetingsResponse struct {
	Meetings []zoomMeetingSummary `json:"meetings"`
}

type zoomMeetingSummary struct {
	ID        int64  `json:"id"`
	UUID      string `json:"uuid"`
	Topic     string `json:"topic"`
	Type      int    `json:"type"`
	StartTime string `json:"start_time"`
	Duration  int    `json:"duration"`
	Timezone  string `json:"timezone"`
	JoinURL   string `json:"join_url"`
	Status    string `json:"status"`
}

type meetingListItem struct {
	ID        int64  `json:"id"`
	UUID      string `json:"uuid"`
	Topic     string `json:"topic"`
	Type      int    `json:"type"`
	StartTime string `json:"start_time,omitempty"`
	Duration  int    `json:"duration"`
	Timezone  string `json:"timezone,omitempty"`
	JoinURL   string `json:"join_url"`
	Status    string `json:"status,omitempty"`
}

func (a *listMeetingsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listMeetingsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.normalize()

	q := url.Values{}
	q.Set("type", params.Type)
	q.Set("page_size", strconv.Itoa(params.PageSize))

	var resp zoomMeetingsResponse
	meetingsURL := a.conn.baseURL + "/users/me/meetings?" + q.Encode()
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, meetingsURL, nil, &resp); err != nil {
		return nil, err
	}

	meetings := make([]meetingListItem, 0, len(resp.Meetings))
	for _, m := range resp.Meetings {
		meetings = append(meetings, meetingListItem{
			ID:        m.ID,
			UUID:      m.UUID,
			Topic:     m.Topic,
			Type:      m.Type,
			StartTime: m.StartTime,
			Duration:  m.Duration,
			Timezone:  m.Timezone,
			JoinURL:   m.JoinURL,
			Status:    m.Status,
		})
	}

	return connectors.JSONResult(map[string]any{
		"total_meetings": len(meetings),
		"meetings":       meetings,
	})
}
