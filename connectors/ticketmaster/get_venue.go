package ticketmaster

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getVenueAction implements connectors.Action for ticketmaster.get_venue.
type getVenueAction struct {
	conn *TicketmasterConnector
}

type getVenueParams struct {
	VenueID string `json:"venue_id"`
	Locale  string `json:"locale"`
}

func (a *getVenueAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getVenueParams
	if err := parseParams(req.Parameters, &params); err != nil {
		return nil, err
	}
	if err := validateTicketmasterID(trimString(params.VenueID), "venue_id"); err != nil {
		return nil, err
	}

	q := url.Values{}
	appendNonEmpty(q, "locale", trimString(params.Locale))

	path := fmt.Sprintf("venues/%s.json", url.PathEscape(trimString(params.VenueID)))
	var out json.RawMessage
	if err := a.conn.doGET(ctx, req.Credentials, path, q, &out); err != nil {
		return nil, err
	}
	return &connectors.ActionResult{Data: out}, nil
}
