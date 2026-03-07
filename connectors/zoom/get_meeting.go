package zoom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getMeetingAction implements connectors.Action for zoom.get_meeting.
// It retrieves meeting details via GET /meetings/{meetingId}.
type getMeetingAction struct {
	conn *ZoomConnector
}

type getMeetingParams struct {
	MeetingID string `json:"meeting_id"`
}

func (p *getMeetingParams) validate() error {
	if p.MeetingID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: meeting_id"}
	}
	return nil
}

// zoomMeetingDetail is the Zoom API response from GET /meetings/{meetingId}.
type zoomMeetingDetail struct {
	ID        int64              `json:"id"`
	UUID      string             `json:"uuid"`
	Topic     string             `json:"topic"`
	Type      int                `json:"type"`
	StartTime string             `json:"start_time"`
	Duration  int                `json:"duration"`
	Timezone  string             `json:"timezone"`
	Agenda    string             `json:"agenda"`
	JoinURL   string             `json:"join_url"`
	StartURL  string             `json:"start_url"`
	Password  string             `json:"password"`
	Status    string             `json:"status"`
	Settings  zoomMeetingSettings `json:"settings"`
	DialIn    []zoomDialInNumber `json:"dial_in_numbers,omitempty"`
}

type zoomMeetingSettings struct {
	JoinBeforeHost bool `json:"join_before_host"`
	WaitingRoom    bool `json:"waiting_room"`
	Mute           bool `json:"mute_upon_entry"`
}

type zoomDialInNumber struct {
	Country string `json:"country"`
	Number  string `json:"number"`
}

func (a *getMeetingAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getMeetingParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var resp zoomMeetingDetail
	meetingURL := a.conn.baseURL + "/meetings/" + url.PathEscape(params.MeetingID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, meetingURL, nil, &resp); err != nil {
		return nil, err
	}

	// Note: start_url is intentionally excluded — it contains a token that
	// grants host privileges and should not be exposed to agents.
	return connectors.JSONResult(map[string]any{
		"id":         resp.ID,
		"uuid":       resp.UUID,
		"topic":      resp.Topic,
		"type":       resp.Type,
		"start_time": resp.StartTime,
		"duration":   resp.Duration,
		"timezone":   resp.Timezone,
		"agenda":     resp.Agenda,
		"join_url":   resp.JoinURL,
		"password":   resp.Password,
		"status":     resp.Status,
		"settings": map[string]any{
			"join_before_host": resp.Settings.JoinBeforeHost,
			"waiting_room":     resp.Settings.WaitingRoom,
			"mute_upon_entry":  resp.Settings.Mute,
		},
		"dial_in_numbers": resp.DialIn,
	})
}
