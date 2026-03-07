package zoom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// deleteMeetingAction implements connectors.Action for zoom.delete_meeting.
// It deletes a meeting via DELETE /meetings/{meetingId}.
type deleteMeetingAction struct {
	conn *ZoomConnector
}

type deleteMeetingParams struct {
	MeetingID            string `json:"meeting_id"`
	ScheduleForReminder  *bool  `json:"schedule_for_reminder,omitempty"`
}

func (p *deleteMeetingParams) validate() error {
	if p.MeetingID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: meeting_id"}
	}
	return nil
}

func (a *deleteMeetingAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteMeetingParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	meetingURL := a.conn.baseURL + "/meetings/" + params.MeetingID
	if params.ScheduleForReminder != nil {
		q := url.Values{}
		if *params.ScheduleForReminder {
			q.Set("schedule_for_reminder", "true")
		} else {
			q.Set("schedule_for_reminder", "false")
		}
		meetingURL += "?" + q.Encode()
	}

	// Zoom returns 204 No Content on successful delete.
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodDelete, meetingURL, nil, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"meeting_id": params.MeetingID,
		"deleted":    true,
	})
}
