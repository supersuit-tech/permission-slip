package zoom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// updateMeetingAction implements connectors.Action for zoom.update_meeting.
// It updates a meeting via PATCH /meetings/{meetingId}.
type updateMeetingAction struct {
	conn *ZoomConnector
}

type updateMeetingParams struct {
	MeetingID string                 `json:"meeting_id"`
	Topic     string                 `json:"topic,omitempty"`
	StartTime string                 `json:"start_time,omitempty"`
	Duration  int                    `json:"duration,omitempty"`
	Timezone  string                 `json:"timezone,omitempty"`
	Agenda    string                 `json:"agenda,omitempty"`
	Settings  map[string]any `json:"settings,omitempty"`
}

func (p *updateMeetingParams) validate() error {
	if p.MeetingID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: meeting_id"}
	}
	return nil
}

// zoomUpdateMeetingRequest is the body sent to PATCH /meetings/{meetingId}.
type zoomUpdateMeetingRequest struct {
	Topic     string                 `json:"topic,omitempty"`
	StartTime string                 `json:"start_time,omitempty"`
	Duration  int                    `json:"duration,omitempty"`
	Timezone  string                 `json:"timezone,omitempty"`
	Agenda    string                 `json:"agenda,omitempty"`
	Settings  map[string]any `json:"settings,omitempty"`
}

func (a *updateMeetingAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateMeetingParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := zoomUpdateMeetingRequest{
		Topic:     params.Topic,
		StartTime: params.StartTime,
		Duration:  params.Duration,
		Timezone:  params.Timezone,
		Agenda:    params.Agenda,
		Settings:  params.Settings,
	}

	// Zoom returns 204 No Content on successful update.
	meetingURL := a.conn.baseURL + "/meetings/" + url.PathEscape(params.MeetingID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPatch, meetingURL, body, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"meeting_id": params.MeetingID,
		"updated":    true,
	})
}
