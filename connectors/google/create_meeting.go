package google

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createMeetingAction implements connectors.Action for google.create_meeting.
// It creates a Google Calendar event with an auto-generated Google Meet
// conference link via POST /calendars/{calendarId}/events?conferenceDataVersion=1.
// API docs: https://developers.google.com/calendar/api/v3/reference/events/insert
// Meet integration: https://developers.google.com/calendar/api/guides/create-events#conferencing
// Required scope: calendar.events
type createMeetingAction struct {
	conn *GoogleConnector
}

// ParameterAliases maps common agent shorthand to the canonical parameter names.
// Agents sometimes send "start"/"end" instead of "start_time"/"end_time".
func (a *createMeetingAction) ParameterAliases() map[string]string {
	return map[string]string{
		"start": "start_time",
		"end":   "end_time",
	}
}

// createMeetingParams is the user-facing parameter schema.
type createMeetingParams struct {
	Summary     string   `json:"summary"`
	Description string   `json:"description"`
	StartTime   string   `json:"start_time"`
	EndTime     string   `json:"end_time"`
	Attendees   []string `json:"attendees"`
	CalendarID  string   `json:"calendar_id"`
}

func (p *createMeetingParams) validate() error {
	if p.Summary == "" {
		return &connectors.ValidationError{Message: "missing required parameter: summary"}
	}
	if p.StartTime == "" {
		return &connectors.ValidationError{Message: "missing required parameter: start_time"}
	}
	if p.EndTime == "" {
		return &connectors.ValidationError{Message: "missing required parameter: end_time"}
	}
	return validateTimeRange(p.StartTime, p.EndTime)
}

func (p *createMeetingParams) normalize() {
	if p.CalendarID == "" {
		p.CalendarID = "primary"
	}
}

// meetingEventRequest is the Google Calendar API request body for creating
// an event with a Google Meet conference.
type meetingEventRequest struct {
	Summary        string                `json:"summary"`
	Description    string                `json:"description,omitempty"`
	Start          calendarEventDateTime `json:"start"`
	End            calendarEventDateTime `json:"end"`
	Attendees      []calendarAttendee    `json:"attendees,omitempty"`
	ConferenceData meetingConferenceData `json:"conferenceData"`
}

// meetingConferenceData requests automatic Google Meet link generation.
type meetingConferenceData struct {
	CreateRequest meetingCreateRequest `json:"createRequest"`
}

type meetingCreateRequest struct {
	RequestID             string                      `json:"requestId"`
	ConferenceSolutionKey meetingConferenceSolutionKey `json:"conferenceSolutionKey"`
}

type meetingConferenceSolutionKey struct {
	Type string `json:"type"`
}

// meetingEventResponse is the Google Calendar API response with conference data.
type meetingEventResponse struct {
	ID             string                        `json:"id"`
	HTMLLink       string                        `json:"htmlLink"`
	Summary        string                        `json:"summary"`
	Status         string                        `json:"status"`
	ConferenceData *meetingConferenceDataResponse `json:"conferenceData,omitempty"`
}

type meetingConferenceDataResponse struct {
	EntryPoints []meetingEntryPoint `json:"entryPoints"`
}

type meetingEntryPoint struct {
	EntryPointType string `json:"entryPointType"`
	URI            string `json:"uri"`
}

// Execute creates a Google Calendar event with an attached Google Meet link.
func (a *createMeetingAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createMeetingParams
	if err := json.Unmarshal(normalizeCalendarTimeParams(req.Parameters), &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.normalize()

	body := meetingEventRequest{
		Summary:     params.Summary,
		Description: params.Description,
		Start:       calendarEventDateTime{DateTime: params.StartTime},
		End:         calendarEventDateTime{DateTime: params.EndTime},
		ConferenceData: meetingConferenceData{
			CreateRequest: meetingCreateRequest{
				RequestID:             meetingRequestID(params.Summary, params.StartTime),
				ConferenceSolutionKey: meetingConferenceSolutionKey{Type: "hangoutsMeet"},
			},
		},
	}
	body.Attendees = buildAttendees(params.Attendees)

	var resp meetingEventResponse
	calURL := a.conn.calendarBaseURL + "/calendars/" + url.PathEscape(params.CalendarID) + "/events?conferenceDataVersion=1"
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, calURL, body, &resp); err != nil {
		return nil, err
	}

	result := map[string]string{
		"id":        resp.ID,
		"html_link": resp.HTMLLink,
		"status":    resp.Status,
	}

	// Extract the Google Meet link from conference entry points.
	if resp.ConferenceData != nil {
		for _, ep := range resp.ConferenceData.EntryPoints {
			if ep.EntryPointType == "video" {
				result["meet_link"] = ep.URI
				break
			}
		}
	}

	return connectors.JSONResult(result)
}

// meetingRequestID generates a deterministic conference request ID from the
// meeting summary and start time. This makes the request idempotent — if the
// same meeting is created twice with the same parameters, the Google API will
// return the same conference link instead of creating a duplicate.
func meetingRequestID(summary, startTime string) string {
	h := sha256.New()
	h.Write([]byte(summary))
	h.Write([]byte(startTime))
	return "meet-" + hex.EncodeToString(h.Sum(nil))[:16]
}
