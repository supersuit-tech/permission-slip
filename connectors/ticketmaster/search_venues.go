package ticketmaster

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchVenuesAction implements connectors.Action for ticketmaster.search_venues.
type searchVenuesAction struct {
	conn *TicketmasterConnector
}

type searchVenuesParams struct {
	Keyword     string `json:"keyword"`
	LatLong     string `json:"latlong"`
	Radius      string `json:"radius"`
	Unit        string `json:"unit"`
	CountryCode string `json:"country_code"`
	StateCode   string `json:"state_code"`
	City        string `json:"city"`
	PostalCode  string `json:"postal_code"`
	Source      string `json:"source"`
	Sort        string `json:"sort"`
	Size        int    `json:"size"`
	Page        int    `json:"page"`
}

func (p *searchVenuesParams) validate() error {
	if trimString(p.Keyword) == "" &&
		trimString(p.City) == "" &&
		trimString(p.PostalCode) == "" &&
		trimString(p.LatLong) == "" {
		return &connectors.ValidationError{Message: "provide at least one of: keyword, city, postal_code, or latlong"}
	}
	if p.Size < 0 || p.Size > 200 {
		return &connectors.ValidationError{Message: "size must be between 0 and 200 (0 uses API default)"}
	}
	if p.Page < 0 {
		return &connectors.ValidationError{Message: "page must be zero or positive"}
	}
	if p.Unit != "" && p.Unit != "miles" && p.Unit != "km" {
		return &connectors.ValidationError{Message: "unit must be \"miles\" or \"km\" when set"}
	}
	return nil
}

func (a *searchVenuesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchVenuesParams
	if err := parseParams(req.Parameters, &params); err != nil {
		return nil, err
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	q := url.Values{}
	appendNonEmpty(q, "keyword", trimString(params.Keyword))
	appendNonEmpty(q, "latlong", trimString(params.LatLong))
	appendNonEmpty(q, "radius", trimString(params.Radius))
	appendNonEmpty(q, "unit", trimString(params.Unit))
	appendNonEmpty(q, "countryCode", trimString(params.CountryCode))
	appendNonEmpty(q, "stateCode", trimString(params.StateCode))
	appendNonEmpty(q, "city", trimString(params.City))
	appendNonEmpty(q, "postalCode", trimString(params.PostalCode))
	appendNonEmpty(q, "source", trimString(params.Source))
	appendNonEmpty(q, "sort", trimString(params.Sort))
	if params.Size > 0 {
		q.Set("size", strconv.Itoa(params.Size))
	}
	if params.Page > 0 {
		q.Set("page", strconv.Itoa(params.Page))
	}

	var out json.RawMessage
	if err := a.conn.doGET(ctx, req.Credentials, "venues.json", q, &out); err != nil {
		return nil, err
	}
	return &connectors.ActionResult{Data: out}, nil
}
