package ticketmaster

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getEventAction implements connectors.Action for ticketmaster.get_event.
type getEventAction struct {
	conn *TicketmasterConnector
}

type getEventParams struct {
	EventID string `json:"event_id"`
	Locale  string `json:"locale"`
}

func (a *getEventAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getEventParams
	if err := parseParams(req.Parameters, &params); err != nil {
		return nil, err
	}
	if err := validateTicketmasterID(trimString(params.EventID), "event_id"); err != nil {
		return nil, err
	}

	q := url.Values{}
	appendNonEmpty(q, "locale", trimString(params.Locale))

	path := fmt.Sprintf("events/%s.json", url.PathEscape(trimString(params.EventID)))
	var out json.RawMessage
	if err := a.conn.doGET(ctx, req.Credentials, path, q, &out); err != nil {
		return nil, err
	}
	return &connectors.ActionResult{Data: out}, nil
}
