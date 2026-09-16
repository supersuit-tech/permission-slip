package google

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

// listCalendarEventsAction implements connectors.Action for google.list_calendar_events.
// It lists events via the Google Calendar API GET /calendars/{calendarId}/events.
type listCalendarEventsAction struct {
	conn *GoogleConnector
}

// listCalendarEventsParams is the user-facing parameter schema.
type listCalendarEventsParams struct {
	CalendarID   string `json:"calendar_id"`
	MaxResults   int    `json:"max_results"`
	TimeMin      string `json:"time_min"`
	TimeMax      string `json:"time_max"`
	SingleEvents *bool  `json:"single_events"`
}

func (p *listCalendarEventsParams) expandInstances() bool {
	return p.SingleEvents == nil || *p.SingleEvents
}

func (p *listCalendarEventsParams) normalize() {
	if p.CalendarID == "" {
		p.CalendarID = "primary"
	}
	if p.MaxResults <= 0 {
		p.MaxResults = 10
	}
	if p.MaxResults > 250 {
		p.MaxResults = 250
	}
}

func (p *listCalendarEventsParams) validate() error {
	var tMin, tMax time.Time
	var err error

	if p.TimeMin != "" {
		tMin, err = time.Parse(time.RFC3339, p.TimeMin)
		if err != nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("time_min must be RFC 3339 format: %v", err)}
		}
	}
	if p.TimeMax != "" {
		tMax, err = time.Parse(time.RFC3339, p.TimeMax)
		if err != nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("time_max must be RFC 3339 format: %v", err)}
		}
	}
	if p.TimeMin != "" && p.TimeMax != "" && !tMax.After(tMin) {
		return &connectors.ValidationError{Message: "time_max must be after time_min"}
	}
	return nil
}

// calendarListResponse is the Google Calendar API response from events.list.
type calendarListResponse struct {
	Items []calendarListItem `json:"items"`
}

type calendarListItem struct {
	ID               string                 `json:"id"`
	Summary          string                 `json:"summary"`
	Description      string                 `json:"description"`
	Status           string                 `json:"status"`
	HTMLLink         string                 `json:"htmlLink"`
	Start            calendarListDateTime   `json:"start"`
	End              calendarListDateTime   `json:"end"`
	Attendees        []calendarListAttendee `json:"attendees"`
	Created          string                 `json:"created"`
	Updated          string                 `json:"updated"`
	RecurringEventID string                 `json:"recurringEventId"`
	OriginalStart    calendarListDateTime   `json:"originalStartTime"`
	Recurrence       []string               `json:"recurrence"`
}

type calendarListDateTime struct {
	DateTime string `json:"dateTime"`
	Date     string `json:"date"`
}

type calendarListAttendee struct {
	Email          string `json:"email"`
	ResponseStatus string `json:"responseStatus"`
}

// eventSummary is the shape returned to the agent.
type eventSummary struct {
	ID                string   `json:"id"`
	Summary           string   `json:"summary"`
	Description       string   `json:"description,omitempty"`
	StartTime         string   `json:"start_time"`
	EndTime           string   `json:"end_time"`
	Status            string   `json:"status"`
	HTMLLink          string   `json:"html_link"`
	Attendees         []string `json:"attendees,omitempty"`
	EventKind         string   `json:"event_kind"`
	RecurringEventID  string   `json:"recurring_event_id,omitempty"`
	OriginalStartTime string   `json:"original_start_time,omitempty"`
	Recurrence        []string `json:"recurrence,omitempty"`
}

func listDateTimeDisplay(dt calendarListDateTime) string {
	if dt.DateTime != "" {
		return dt.DateTime
	}
	return dt.Date
}

func classifyListedEvent(item calendarListItem) string {
	if item.RecurringEventID != "" {
		return calendarEventKindInstance
	}
	if len(item.Recurrence) > 0 {
		return calendarEventKindSeriesMaster
	}
	return calendarEventKindSingle
}

// Execute lists upcoming events from Google Calendar.
func (a *listCalendarEventsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listCalendarEventsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.normalize()

	q := url.Values{}
	q.Set("maxResults", strconv.Itoa(params.MaxResults))
	if params.expandInstances() {
		q.Set("singleEvents", "true")
		q.Set("orderBy", "startTime")
	} else {
		q.Set("singleEvents", "false")
	}

	if params.TimeMin != "" {
		q.Set("timeMin", params.TimeMin)
	} else {
		q.Set("timeMin", time.Now().UTC().Format(time.RFC3339))
	}
	if params.TimeMax != "" {
		q.Set("timeMax", params.TimeMax)
	}

	var listResp calendarListResponse
	listURL := a.conn.calendarBaseURL + "/calendars/" + url.PathEscape(params.CalendarID) + "/events?" + q.Encode()
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, listURL, nil, &listResp); err != nil {
		return nil, err
	}

	events := make([]eventSummary, 0, len(listResp.Items))
	for _, item := range listResp.Items {
		ev := eventSummary{
			ID:          item.ID,
			Summary:     item.Summary,
			Description: item.Description,
			Status:      item.Status,
			HTMLLink:    item.HTMLLink,
			EventKind:   classifyListedEvent(item),
		}
		// Prefer dateTime, fall back to date for all-day events.
		if item.Start.DateTime != "" {
			ev.StartTime = item.Start.DateTime
		} else {
			ev.StartTime = item.Start.Date
		}
		if item.End.DateTime != "" {
			ev.EndTime = item.End.DateTime
		} else {
			ev.EndTime = item.End.Date
		}
		if item.RecurringEventID != "" {
			ev.RecurringEventID = item.RecurringEventID
		}
		if orig := listDateTimeDisplay(item.OriginalStart); orig != "" {
			ev.OriginalStartTime = orig
		}
		if len(item.Recurrence) > 0 {
			ev.Recurrence = item.Recurrence
		}
		attendees := make([]string, 0, len(item.Attendees))
		for _, att := range item.Attendees {
			attendees = append(attendees, att.Email)
		}
		if len(attendees) > 0 {
			ev.Attendees = attendees
		}
		events = append(events, ev)
	}

	return connectors.JSONResult(map[string]any{
		"events": events,
	})
}
