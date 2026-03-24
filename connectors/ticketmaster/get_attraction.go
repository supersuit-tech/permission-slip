package ticketmaster

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getAttractionAction implements connectors.Action for ticketmaster.get_attraction.
type getAttractionAction struct {
	conn *TicketmasterConnector
}

type getAttractionParams struct {
	AttractionID string `json:"attraction_id"`
	Locale       string `json:"locale"`
}

func (a *getAttractionAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getAttractionParams
	if err := parseParams(req.Parameters, &params); err != nil {
		return nil, err
	}
	if err := validateTicketmasterID(trimString(params.AttractionID), "attraction_id"); err != nil {
		return nil, err
	}

	q := url.Values{}
	appendNonEmpty(q, "locale", trimString(params.Locale))

	path := fmt.Sprintf("attractions/%s.json", url.PathEscape(trimString(params.AttractionID)))
	var out json.RawMessage
	if err := a.conn.doGET(ctx, req.Credentials, path, q, &out); err != nil {
		return nil, err
	}
	return &connectors.ActionResult{Data: out}, nil
}
