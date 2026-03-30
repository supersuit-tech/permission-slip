package calendly

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listScheduledEventsAction implements connectors.Action for calendly.list_scheduled_events.
// It lists scheduled events via GET /scheduled_events?user={user_uri}.
type listScheduledEventsAction struct {
	conn *CalendlyConnector
}

type listScheduledEventsParams struct {
	UserURI      string `json:"user_uri,omitempty"`
	MinStartTime string `json:"min_start_time"`
	MaxStartTime string `json:"max_start_time"`
	Status       string `json:"status"`
	Count        int    `json:"count"`
}

var validEventStatuses = map[string]bool{
	"active":   true,
	"canceled": true,
}

func (p *listScheduledEventsParams) validate() error {
	if p.Status != "" && !validEventStatuses[p.Status] {
		return &connectors.ValidationError{Message: fmt.Sprintf("status must be one of: active, canceled; got %q", p.Status)}
	}
	return nil
}

func (p *listScheduledEventsParams) normalize() {
	if p.Count <= 0 {
		p.Count = 20
	}
	if p.Count > 100 {
		p.Count = 100
	}
}

// calendlyScheduledEventsResponse is the Calendly API response from GET /scheduled_events.
type calendlyScheduledEventsResponse struct {
	Collection []calendlyScheduledEvent `json:"collection"`
}

type calendlyScheduledEvent struct {
	URI              string           `json:"uri"`
	Name             string           `json:"name"`
	Status           string           `json:"status"`
	StartTime        string           `json:"start_time"`
	EndTime          string           `json:"end_time"`
	EventType        string           `json:"event_type"`
	Location         calendlyLocation `json:"location"`
	CreatedAt        string           `json:"created_at"`
	UpdatedAt        string           `json:"updated_at"`
	EventMemberships []map[string]any `json:"event_memberships"`
	EventGuests      []calendlyGuest  `json:"event_guests"`
}

type scheduledEventItem struct {
	URI        string   `json:"uri"`
	Name       string   `json:"name"`
	Status     string   `json:"status"`
	StartTime  string   `json:"start_time"`
	EndTime    string   `json:"end_time"`
	EventType  string   `json:"event_type"`
	Location   string   `json:"location,omitempty"`
	GuestCount int      `json:"guest_count"`
	Guests     []string `json:"guests,omitempty"`
}

func (a *listScheduledEventsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listScheduledEventsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.normalize()

	userURI, err := a.conn.resolveUserURI(ctx, req.Credentials, params.UserURI)
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("user", userURI)
	q.Set("count", strconv.Itoa(params.Count))
	if params.MinStartTime != "" {
		q.Set("min_start_time", params.MinStartTime)
	}
	if params.MaxStartTime != "" {
		q.Set("max_start_time", params.MaxStartTime)
	}
	if params.Status != "" {
		q.Set("status", params.Status)
	}

	var resp calendlyScheduledEventsResponse
	reqURL := a.conn.baseURL + "/scheduled_events?" + q.Encode()
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, reqURL, nil, &resp); err != nil {
		return nil, err
	}

	items := make([]scheduledEventItem, 0, len(resp.Collection))
	for _, ev := range resp.Collection {
		item := scheduledEventItem{
			URI:        ev.URI,
			Name:       ev.Name,
			Status:     ev.Status,
			StartTime:  ev.StartTime,
			EndTime:    ev.EndTime,
			EventType:  ev.EventType,
			Location:   ev.Location.Location,
			GuestCount: len(ev.EventGuests),
		}
		for _, g := range ev.EventGuests {
			item.Guests = append(item.Guests, g.Email)
		}
		items = append(items, item)
	}

	return connectors.JSONResult(map[string]any{
		"total_events": len(items),
		"events":       items,
	})
}
