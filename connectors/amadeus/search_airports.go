package amadeus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchAirportsAction implements connectors.Action for amadeus.search_airports.
// It looks up airports by name or IATA code via GET /v1/reference-data/locations.
type searchAirportsAction struct {
	conn *AmadeusConnector
}

type searchAirportsParams struct {
	Keyword string `json:"keyword"`
	SubType string `json:"subtype"`
}

func (p *searchAirportsParams) validate() error {
	if p.Keyword == "" {
		return &connectors.ValidationError{Message: "missing required parameter: keyword"}
	}
	if p.SubType != "" && p.SubType != "AIRPORT" && p.SubType != "CITY" {
		return &connectors.ValidationError{Message: "subtype must be AIRPORT or CITY"}
	}
	return nil
}

func (a *searchAirportsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchAirportsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	subType := params.SubType
	if subType == "" {
		subType = "AIRPORT,CITY"
	}

	q := url.Values{
		"subType": {subType},
		"keyword": {params.Keyword},
	}

	path := "/v1/reference-data/locations?" + q.Encode()

	var resp struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
