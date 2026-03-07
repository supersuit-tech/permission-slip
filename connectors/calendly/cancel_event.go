package calendly

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// cancelEventAction implements connectors.Action for calendly.cancel_event.
// It cancels an event via POST /scheduled_events/{uuid}/cancellation.
type cancelEventAction struct {
	conn *CalendlyConnector
}

type cancelEventParams struct {
	EventUUID string `json:"event_uuid"`
	Reason    string `json:"reason"`
}

func (p *cancelEventParams) validate() error {
	if p.EventUUID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: event_uuid"}
	}
	if err := validateUUID(p.EventUUID); err != nil {
		return err
	}
	return nil
}

type calendlyCancelRequest struct {
	Reason string `json:"reason,omitempty"`
}

type calendlyCancelResponse struct {
	Resource struct {
		CanceledBy string `json:"canceled_by"`
		Reason     string `json:"reason"`
	} `json:"resource"`
}

func (a *cancelEventAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params cancelEventParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := calendlyCancelRequest{
		Reason: params.Reason,
	}

	var resp calendlyCancelResponse
	reqURL := a.conn.baseURL + "/scheduled_events/" + url.PathEscape(params.EventUUID) + "/cancellation"
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, reqURL, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"canceled_by": resp.Resource.CanceledBy,
		"reason":      resp.Resource.Reason,
	})
}
