package calendly

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getEventAction implements connectors.Action for calendly.get_event.
// It gets event details via GET /scheduled_events/{uuid}.
type getEventAction struct {
	conn *CalendlyConnector
}

type getEventParams struct {
	EventUUID string `json:"event_uuid"`
}

func (p *getEventParams) validate() error {
	if p.EventUUID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: event_uuid"}
	}
	if err := validateUUID(p.EventUUID); err != nil {
		return err
	}
	return nil
}

type calendlyGetEventResponse struct {
	Resource struct {
		URI       string `json:"uri"`
		Name      string `json:"name"`
		Status    string `json:"status"`
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
		EventType string `json:"event_type"`
		Location  struct {
			Type     string `json:"type"`
			Location string `json:"location"`
			JoinURL  string `json:"join_url"`
		} `json:"location"`
		CreatedAt        string                   `json:"created_at"`
		UpdatedAt        string                   `json:"updated_at"`
		EventMemberships []map[string]any         `json:"event_memberships"`
		EventGuests      []calendlyScheduledGuest `json:"event_guests"`
	} `json:"resource"`
}

func (a *getEventAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getEventParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var resp calendlyGetEventResponse
	reqURL := a.conn.baseURL + "/scheduled_events/" + url.PathEscape(params.EventUUID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, reqURL, nil, &resp); err != nil {
		return nil, err
	}

	result := map[string]any{
		"uri":        resp.Resource.URI,
		"name":       resp.Resource.Name,
		"status":     resp.Resource.Status,
		"start_time": resp.Resource.StartTime,
		"end_time":   resp.Resource.EndTime,
		"event_type": resp.Resource.EventType,
		"created_at": resp.Resource.CreatedAt,
		"updated_at": resp.Resource.UpdatedAt,
	}
	if resp.Resource.Location.Type != "" {
		result["location_type"] = resp.Resource.Location.Type
	}
	if resp.Resource.Location.Location != "" {
		result["location"] = resp.Resource.Location.Location
	}
	if resp.Resource.Location.JoinURL != "" {
		result["join_url"] = resp.Resource.Location.JoinURL
	}
	if len(resp.Resource.EventGuests) > 0 {
		guests := make([]map[string]string, 0, len(resp.Resource.EventGuests))
		for _, g := range resp.Resource.EventGuests {
			guests = append(guests, map[string]string{
				"email": g.Email,
			})
		}
		result["guests"] = guests
	}

	return connectors.JSONResult(result)
}
