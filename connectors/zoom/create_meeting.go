package zoom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createMeetingAction implements connectors.Action for zoom.create_meeting.
// It creates a meeting via POST /users/me/meetings.
type createMeetingAction struct {
	conn *ZoomConnector
}

type createMeetingParams struct {
	Topic     string                 `json:"topic"`
	Type      int                    `json:"type"`
	StartTime string                 `json:"start_time,omitempty"`
	Duration  int                    `json:"duration,omitempty"`
	Timezone  string                 `json:"timezone,omitempty"`
	Agenda    string                 `json:"agenda,omitempty"`
	Settings  map[string]interface{} `json:"settings,omitempty"`
}

func (p *createMeetingParams) validate() error {
	if p.Topic == "" {
		return &connectors.ValidationError{Message: "missing required parameter: topic"}
	}
	if p.Type != 0 && p.Type != 1 && p.Type != 2 {
		return &connectors.ValidationError{Message: "type must be 1 (instant) or 2 (scheduled)"}
	}
	return nil
}

func (p *createMeetingParams) normalize() {
	if p.Type == 0 {
		p.Type = 2 // default to scheduled
	}
}

// zoomCreateMeetingRequest is sent to the Zoom API.
type zoomCreateMeetingRequest struct {
	Topic     string                 `json:"topic"`
	Type      int                    `json:"type"`
	StartTime string                 `json:"start_time,omitempty"`
	Duration  int                    `json:"duration,omitempty"`
	Timezone  string                 `json:"timezone,omitempty"`
	Agenda    string                 `json:"agenda,omitempty"`
	Settings  map[string]interface{} `json:"settings,omitempty"`
}

// zoomCreateMeetingResponse is the Zoom API response from POST /users/me/meetings.
type zoomCreateMeetingResponse struct {
	ID        int64  `json:"id"`
	UUID      string `json:"uuid"`
	Topic     string `json:"topic"`
	Type      int    `json:"type"`
	StartTime string `json:"start_time"`
	Duration  int    `json:"duration"`
	Timezone  string `json:"timezone"`
	Agenda    string `json:"agenda"`
	JoinURL   string `json:"join_url"`
	StartURL  string `json:"start_url"`
	Password  string `json:"password"`
}

func (a *createMeetingAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createMeetingParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.normalize()

	body := zoomCreateMeetingRequest{
		Topic:     params.Topic,
		Type:      params.Type,
		StartTime: params.StartTime,
		Duration:  params.Duration,
		Timezone:  params.Timezone,
		Agenda:    params.Agenda,
		Settings:  params.Settings,
	}

	var resp zoomCreateMeetingResponse
	meetingURL := a.conn.baseURL + "/users/me/meetings"
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, meetingURL, body, &resp); err != nil {
		return nil, err
	}

	result := map[string]any{
		"id":         resp.ID,
		"uuid":       resp.UUID,
		"topic":      resp.Topic,
		"type":       resp.Type,
		"start_time": resp.StartTime,
		"duration":   resp.Duration,
		"timezone":   resp.Timezone,
		"join_url":   resp.JoinURL,
		"start_url":  resp.StartURL,
		"password":   resp.Password,
	}
	if resp.Agenda != "" {
		result["agenda"] = resp.Agenda
	}

	return connectors.JSONResult(result)
}
