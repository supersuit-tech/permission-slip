package google

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// Recurring-event edit/delete scope. Matches the Calendar UI choices:
// one occurrence, the entire series, or this instance and all following.
const (
	calendarScopeInstance         = "instance"
	calendarScopeSeries           = "series"
	calendarScopeThisAndFollowing = "this_and_following"
)

const (
	calendarEventKindSingle       = "single"
	calendarEventKindSeriesMaster = "series_master"
	calendarEventKindInstance     = "instance"
)

// instanceIDTimedSuffix matches Google's expanded instance id
// `{masterId}_{YYYYMMDDTHHMMSSZ}`.
var instanceIDTimedSuffix = regexp.MustCompile(`_[0-9]{8}T[0-9]{6}Z$`)

// instanceIDAllDaySuffix matches `{masterId}_{YYYYMMDD}` for all-day instances.
var instanceIDAllDaySuffix = regexp.MustCompile(`_[0-9]{8}$`)

// calendarEventResource is the Calendar API event subset used to classify
// masters vs instances and to split a series (this_and_following).
type calendarEventResource struct {
	ID               string                `json:"id"`
	Summary          string                `json:"summary"`
	Description      string                `json:"description"`
	Location         string                `json:"location"`
	Status           string                `json:"status"`
	HTMLLink         string                `json:"htmlLink"`
	Updated          string                `json:"updated"`
	Recurrence       []string              `json:"recurrence"`
	RecurringEventID string                `json:"recurringEventId"`
	Start            calendarEventDateTime `json:"start"`
	End              calendarEventDateTime `json:"end"`
	OriginalStart    calendarEventDateTime `json:"originalStartTime"`
	Attendees        []calendarAttendee    `json:"attendees"`
}

type calendarInstancesResponse struct {
	Items []calendarEventResource `json:"items"`
}

func parseCalendarEventScope(scope string) (string, error) {
	switch scope {
	case "", calendarScopeInstance, calendarScopeSeries, calendarScopeThisAndFollowing:
		return scope, nil
	default:
		return "", &connectors.ValidationError{
			Message: "scope must be instance, series, or this_and_following",
		}
	}
}

func looksLikeInstanceEventID(eventID string) bool {
	return instanceIDTimedSuffix.MatchString(eventID) || instanceIDAllDaySuffix.MatchString(eventID)
}

func classifyCalendarEvent(ev *calendarEventResource) string {
	if ev == nil {
		return calendarEventKindSingle
	}
	if ev.RecurringEventID != "" {
		return calendarEventKindInstance
	}
	if len(ev.Recurrence) > 0 {
		return calendarEventKindSeriesMaster
	}
	return calendarEventKindSingle
}

func (d calendarEventDateTime) display() string {
	if d.DateTime != "" {
		return d.DateTime
	}
	return d.Date
}

func (d calendarEventDateTime) isAllDay() bool {
	return d.Date != "" && d.DateTime == ""
}

func parseInstanceStart(value string) (time.Time, bool, error) {
	if value == "" {
		return time.Time{}, false, &connectors.ValidationError{Message: "missing instance_start"}
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, false, nil
	}
	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t, true, nil
	}
	return time.Time{}, false, &connectors.ValidationError{
		Message: "instance_start must be RFC 3339 or a YYYY-MM-DD date",
	}
}

func sameCalendarInstant(a, b time.Time) bool {
	return a.UTC().Equal(b.UTC())
}

func calendarDateTimesEqual(a, b calendarEventDateTime) bool {
	if a.display() == "" || b.display() == "" {
		return false
	}
	if a.isAllDay() || b.isAllDay() {
		aDate := a.Date
		bDate := b.Date
		if aDate == "" {
			if t, err := time.Parse(time.RFC3339, a.DateTime); err == nil {
				aDate = t.UTC().Format("2006-01-02")
			}
		}
		if bDate == "" {
			if t, err := time.Parse(time.RFC3339, b.DateTime); err == nil {
				bDate = t.UTC().Format("2006-01-02")
			}
		}
		return aDate != "" && aDate == bDate
	}
	ta, errA := time.Parse(time.RFC3339, a.DateTime)
	tb, errB := time.Parse(time.RFC3339, b.DateTime)
	if errA != nil || errB != nil {
		return a.DateTime == b.DateTime
	}
	return sameCalendarInstant(ta, tb)
}

func instanceStartMatches(ev *calendarEventResource, start time.Time, allDay bool) bool {
	candidates := []calendarEventDateTime{ev.OriginalStart, ev.Start}
	want := calendarEventDateTime{}
	if allDay {
		want.Date = start.Format("2006-01-02")
	} else {
		want.DateTime = start.Format(time.RFC3339)
	}
	for _, cand := range candidates {
		if calendarDateTimesEqual(cand, want) {
			return true
		}
	}
	return false
}

func validateScopeAgainstEventID(scope, eventID, instanceStart string) error {
	if _, err := parseCalendarEventScope(scope); err != nil {
		return err
	}
	if instanceStart != "" {
		if _, _, err := parseInstanceStart(instanceStart); err != nil {
			return err
		}
		if scope == calendarScopeSeries {
			return &connectors.ValidationError{
				Message: "instance_start cannot be used with scope=series; pass the series master event_id",
			}
		}
		if scope == "" {
			return &connectors.ValidationError{
				Message: "instance_start requires scope=instance or scope=this_and_following",
			}
		}
	}

	switch scope {
	case calendarScopeInstance:
		if !looksLikeInstanceEventID(eventID) && instanceStart == "" {
			return &connectors.ValidationError{
				Message: "scope=instance requires an expanded instance event_id (from list_calendar_events) or a series master event_id plus instance_start",
			}
		}
	case calendarScopeSeries:
		if looksLikeInstanceEventID(eventID) {
			return &connectors.ValidationError{
				Message: "scope=series requires the series master event_id, not an expanded instance id (use recurring_event_id from list_calendar_events)",
			}
		}
	case calendarScopeThisAndFollowing:
		if !looksLikeInstanceEventID(eventID) && instanceStart == "" {
			return &connectors.ValidationError{
				Message: "scope=this_and_following requires an expanded instance event_id or a series master event_id plus instance_start",
			}
		}
	}
	return nil
}

func calendarEventPath(baseURL, calendarID, eventID string) string {
	return baseURL + "/calendars/" + url.PathEscape(calendarID) + "/events/" + url.PathEscape(eventID)
}

func (c *GoogleConnector) getCalendarEvent(ctx context.Context, creds connectors.Credentials, calendarID, eventID string) (*calendarEventResource, error) {
	var ev calendarEventResource
	getURL := calendarEventPath(c.calendarBaseURL, calendarID, eventID)
	if err := c.doJSON(ctx, creds, http.MethodGet, getURL, nil, &ev); err != nil {
		return nil, err
	}
	if ev.ID == "" {
		return nil, &connectors.ExternalError{Message: "calendar event response missing id"}
	}
	return &ev, nil
}

func (c *GoogleConnector) listCalendarEventInstances(ctx context.Context, creds connectors.Credentials, calendarID, masterID string, timeMin, timeMax time.Time, maxResults int) ([]calendarEventResource, error) {
	if maxResults <= 0 {
		maxResults = 50
	}
	q := url.Values{}
	q.Set("maxResults", strconv.Itoa(maxResults))
	if !timeMin.IsZero() {
		q.Set("timeMin", timeMin.UTC().Format(time.RFC3339))
	}
	if !timeMax.IsZero() {
		q.Set("timeMax", timeMax.UTC().Format(time.RFC3339))
	}
	var resp calendarInstancesResponse
	listURL := calendarEventPath(c.calendarBaseURL, calendarID, masterID) + "/instances?" + q.Encode()
	if err := c.doJSON(ctx, creds, http.MethodGet, listURL, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (c *GoogleConnector) findInstanceByStart(ctx context.Context, creds connectors.Credentials, calendarID, masterID, instanceStart string) (*calendarEventResource, error) {
	start, allDay, err := parseInstanceStart(instanceStart)
	if err != nil {
		return nil, err
	}
	windowEnd := start.Add(24 * time.Hour)
	items, err := c.listCalendarEventInstances(ctx, creds, calendarID, masterID, start, windowEnd, 50)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if instanceStartMatches(&items[i], start, allDay) {
			return &items[i], nil
		}
	}
	return nil, &connectors.ValidationError{
		Message: fmt.Sprintf("no series instance found at instance_start %s", instanceStart),
	}
}

// resolvedCalendarEvent is the event id to PATCH/DELETE plus fetched metadata
// when the Calendar API was consulted (needed for this_and_following splits).
type resolvedCalendarEvent struct {
	TargetID string
	Scope    string
	Kind     string
	Event    *calendarEventResource
	Master   *calendarEventResource
	Instance *calendarEventResource
}

func (c *GoogleConnector) resolveCalendarEventTarget(ctx context.Context, creds connectors.Credentials, calendarID, eventID, scope, instanceStart string) (*resolvedCalendarEvent, error) {
	if err := validateScopeAgainstEventID(scope, eventID, instanceStart); err != nil {
		return nil, err
	}

	resolved := &resolvedCalendarEvent{TargetID: eventID, Scope: scope}

	if scope == "" {
		return resolved, nil
	}

	switch scope {
	case calendarScopeInstance:
		if looksLikeInstanceEventID(eventID) && instanceStart == "" {
			resolved.Kind = calendarEventKindInstance
			return resolved, nil
		}
		if instanceStart != "" {
			masterID := eventID
			if looksLikeInstanceEventID(eventID) {
				ev, err := c.getCalendarEvent(ctx, creds, calendarID, eventID)
				if err != nil {
					return nil, err
				}
				if ev.RecurringEventID != "" {
					masterID = ev.RecurringEventID
				}
			}
			inst, err := c.findInstanceByStart(ctx, creds, calendarID, masterID, instanceStart)
			if err != nil {
				return nil, err
			}
			resolved.TargetID = inst.ID
			resolved.Kind = calendarEventKindInstance
			resolved.Instance = inst
			return resolved, nil
		}
		return resolved, nil

	case calendarScopeSeries:
		resolved.Kind = calendarEventKindSeriesMaster
		return resolved, nil

	case calendarScopeThisAndFollowing:
		return c.resolveThisAndFollowing(ctx, creds, calendarID, eventID, instanceStart)
	}

	return resolved, nil
}

func (c *GoogleConnector) resolveThisAndFollowing(ctx context.Context, creds connectors.Credentials, calendarID, eventID, instanceStart string) (*resolvedCalendarEvent, error) {
	resolved := &resolvedCalendarEvent{
		TargetID: eventID,
		Scope:    calendarScopeThisAndFollowing,
	}

	var instance *calendarEventResource
	var err error
	if looksLikeInstanceEventID(eventID) {
		instance, err = c.getCalendarEvent(ctx, creds, calendarID, eventID)
		if err != nil {
			return nil, err
		}
		if instanceStart != "" {
			start, allDay, parseErr := parseInstanceStart(instanceStart)
			if parseErr != nil {
				return nil, parseErr
			}
			if !instanceStartMatches(instance, start, allDay) {
				return nil, &connectors.ValidationError{
					Message: "instance_start does not match the provided instance event_id",
				}
			}
		}
	} else if instanceStart != "" {
		instance, err = c.findInstanceByStart(ctx, creds, calendarID, eventID, instanceStart)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, &connectors.ValidationError{
			Message: "scope=this_and_following requires an expanded instance event_id or a series master event_id plus instance_start",
		}
	}

	kind := classifyCalendarEvent(instance)
	if kind == calendarEventKindSingle {
		// Non-recurring event: this_and_following is the whole event.
		resolved.TargetID = instance.ID
		resolved.Kind = calendarEventKindSingle
		resolved.Event = instance
		resolved.Instance = instance
		return resolved, nil
	}

	masterID := instance.RecurringEventID
	if masterID == "" {
		if kind == calendarEventKindSeriesMaster {
			masterID = instance.ID
		} else {
			return nil, &connectors.ValidationError{
				Message: "event is not a recurring instance; this_and_following needs a series instance",
			}
		}
	}

	master, err := c.getCalendarEvent(ctx, creds, calendarID, masterID)
	if err != nil {
		return nil, err
	}

	resolved.TargetID = instance.ID
	resolved.Kind = calendarEventKindInstance
	resolved.Event = instance
	resolved.Instance = instance
	resolved.Master = master
	return resolved, nil
}

func isFirstSeriesInstance(master, instance *calendarEventResource) bool {
	if master == nil || instance == nil {
		return false
	}
	anchor := instance.OriginalStart
	if anchor.display() == "" {
		anchor = instance.Start
	}
	return calendarDateTimesEqual(master.Start, anchor)
}

func instanceCutoffTime(instance *calendarEventResource) (time.Time, bool, error) {
	anchor := instance.OriginalStart
	if anchor.display() == "" {
		anchor = instance.Start
	}
	if anchor.isAllDay() {
		t, err := time.Parse("2006-01-02", anchor.Date)
		if err != nil {
			return time.Time{}, true, &connectors.ValidationError{
				Message: fmt.Sprintf("instance original start date is invalid: %v", err),
			}
		}
		return t, true, nil
	}
	if anchor.DateTime == "" {
		return time.Time{}, false, &connectors.ValidationError{
			Message: "instance is missing original start time",
		}
	}
	t, err := time.Parse(time.RFC3339, anchor.DateTime)
	if err != nil {
		return time.Time{}, false, &connectors.ValidationError{
			Message: fmt.Sprintf("instance original start time is invalid: %v", err),
		}
	}
	return t, false, nil
}

func (c *GoogleConnector) truncateSeriesBeforeInstance(ctx context.Context, creds connectors.Credentials, calendarID string, master, instance *calendarEventResource) error {
	cutoff, allDay, err := instanceCutoffTime(instance)
	if err != nil {
		return err
	}
	recurrence, err := endRecurrenceBefore(master.Recurrence, cutoff, allDay)
	if err != nil {
		return err
	}
	patchURL := calendarEventPath(c.calendarBaseURL, calendarID, master.ID)
	return c.doJSON(ctx, creds, http.MethodPatch, patchURL, map[string]any{
		"recurrence": recurrence,
	}, nil)
}

func copyEventDateTime(src calendarEventDateTime) calendarEventDateTime {
	return calendarEventDateTime{
		DateTime: src.DateTime,
		Date:     src.Date,
		TimeZone: src.TimeZone,
	}
}

func attendeeEmails(attendees []calendarAttendee) []string {
	if len(attendees) == 0 {
		return nil
	}
	out := make([]string, 0, len(attendees))
	for _, a := range attendees {
		if a.Email != "" {
			out = append(out, a.Email)
		}
	}
	return out
}

func newSeriesBodyFromMaster(master, instance *calendarEventResource, updates map[string]any, recurrence []string) map[string]any {
	body := map[string]any{
		"summary":     master.Summary,
		"description": master.Description,
		"location":    master.Location,
		"start":       copyEventDateTime(instance.Start),
		"end":         copyEventDateTime(instance.End),
	}
	if emails := attendeeEmails(master.Attendees); len(emails) > 0 {
		body["attendees"] = buildAttendees(emails)
	}
	for k, v := range updates {
		if k == "recurrence" {
			// Recurrence is applied after the merge so a COUNT rewrite
			// from recurrenceForSplitSeries is not overwritten.
			continue
		}
		body[k] = v
	}
	if len(recurrence) > 0 {
		body["recurrence"] = recurrence
	} else if len(master.Recurrence) > 0 {
		body["recurrence"] = master.Recurrence
	}
	return body
}

func remainingCountFromRRULE(recurrence []string) (int, bool) {
	for _, line := range recurrence {
		if !strings.HasPrefix(line, "RRULE:") {
			continue
		}
		for _, part := range strings.Split(strings.TrimPrefix(line, "RRULE:"), ";") {
			if strings.HasPrefix(part, "COUNT=") {
				n, err := strconv.Atoi(strings.TrimPrefix(part, "COUNT="))
				if err != nil || n <= 0 {
					return 0, false
				}
				return n, true
			}
		}
	}
	return 0, false
}

func (c *GoogleConnector) recurrenceForSplitSeries(ctx context.Context, creds connectors.Credentials, calendarID string, master, instance *calendarEventResource, override []string) ([]string, error) {
	recurrence := override
	if len(recurrence) == 0 {
		recurrence = append([]string(nil), master.Recurrence...)
	}
	if _, hasCount := remainingCountFromRRULE(recurrence); !hasCount {
		return recurrence, nil
	}
	cutoff, _, err := instanceCutoffTime(instance)
	if err != nil {
		return nil, err
	}
	items, err := c.listCalendarEventInstances(ctx, creds, calendarID, master.ID, cutoff, time.Time{}, 250)
	if err != nil {
		return nil, err
	}
	remaining := len(items)
	if remaining <= 0 {
		remaining = 1
	}
	return setRecurrenceCount(recurrence, remaining), nil
}
