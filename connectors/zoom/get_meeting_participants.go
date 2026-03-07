package zoom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getMeetingParticipantsAction implements connectors.Action for zoom.get_meeting_participants.
// It retrieves participants via GET /past_meetings/{meetingId}/participants.
type getMeetingParticipantsAction struct {
	conn *ZoomConnector
}

type getMeetingParticipantsParams struct {
	MeetingID string `json:"meeting_id"`
}

func (p *getMeetingParticipantsParams) validate() error {
	if p.MeetingID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: meeting_id"}
	}
	return nil
}

// zoomParticipantsResponse is the Zoom API response from GET /past_meetings/{meetingId}/participants.
type zoomParticipantsResponse struct {
	Participants []zoomParticipant `json:"participants"`
}

type zoomParticipant struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"user_email"`
	JoinTime string `json:"join_time"`
	LeaveTime string `json:"leave_time"`
	Duration int    `json:"duration"`
}

type participantItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email,omitempty"`
	JoinTime  string `json:"join_time"`
	LeaveTime string `json:"leave_time"`
	Duration  int    `json:"duration"`
}

func (a *getMeetingParticipantsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getMeetingParticipantsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var resp zoomParticipantsResponse
	participantsURL := a.conn.baseURL + "/past_meetings/" + params.MeetingID + "/participants"
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, participantsURL, nil, &resp); err != nil {
		return nil, err
	}

	participants := make([]participantItem, 0, len(resp.Participants))
	for _, p := range resp.Participants {
		participants = append(participants, participantItem{
			ID:        p.ID,
			Name:      p.Name,
			Email:     p.Email,
			JoinTime:  p.JoinTime,
			LeaveTime: p.LeaveTime,
			Duration:  p.Duration,
		})
	}

	return connectors.JSONResult(map[string]any{
		"participants": participants,
	})
}
