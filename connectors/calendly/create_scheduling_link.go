package calendly

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createSchedulingLinkAction implements connectors.Action for calendly.create_scheduling_link.
// It creates a scheduling link via POST /scheduling_links.
type createSchedulingLinkAction struct {
	conn *CalendlyConnector
}

type createSchedulingLinkParams struct {
	EventTypeURI  string `json:"event_type_uri"`
	MaxEventCount int    `json:"max_event_count"`
	OwnerType     string `json:"owner_type"`
}

func (p *createSchedulingLinkParams) validate() error {
	if p.EventTypeURI == "" {
		return &connectors.ValidationError{Message: "missing required parameter: event_type_uri"}
	}
	return nil
}

func (p *createSchedulingLinkParams) normalize() {
	if p.MaxEventCount <= 0 {
		p.MaxEventCount = 1
	}
	if p.OwnerType == "" {
		p.OwnerType = "EventType"
	}
}

type calendlySchedulingLinkRequest struct {
	MaxEventCount int    `json:"max_event_count"`
	Owner         string `json:"owner"`
	OwnerType     string `json:"owner_type"`
}

type calendlySchedulingLinkResponse struct {
	Resource struct {
		BookingURL    string `json:"booking_url"`
		Owner         string `json:"owner"`
		OwnerType     string `json:"owner_type"`
	} `json:"resource"`
}

func (a *createSchedulingLinkAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createSchedulingLinkParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.normalize()

	body := calendlySchedulingLinkRequest{
		MaxEventCount: params.MaxEventCount,
		Owner:         params.EventTypeURI,
		OwnerType:     params.OwnerType,
	}

	var resp calendlySchedulingLinkResponse
	reqURL := a.conn.baseURL + "/scheduling_links"
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, reqURL, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"booking_url": resp.Resource.BookingURL,
		"owner":       resp.Resource.Owner,
		"owner_type":  resp.Resource.OwnerType,
	})
}
