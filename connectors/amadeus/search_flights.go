package amadeus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchFlightsAction implements connectors.Action for amadeus.search_flights.
// It searches flight offers via GET /v2/shopping/flight-offers.
type searchFlightsAction struct {
	conn *AmadeusConnector
}

type searchFlightsParams struct {
	Origin        string `json:"origin"`
	Destination   string `json:"destination"`
	DepartureDate string `json:"departure_date"`
	ReturnDate    string `json:"return_date"`
	Adults        int    `json:"adults"`
	Cabin         string `json:"cabin"`
	Nonstop       bool   `json:"nonstop"`
	MaxResults    int    `json:"max_results"`
}

func (p *searchFlightsParams) validate() error {
	if p.Origin == "" {
		return &connectors.ValidationError{Message: "missing required parameter: origin"}
	}
	if !validIATACode(p.Origin) {
		return &connectors.ValidationError{Message: "origin must be a 3-letter IATA code (e.g., SFO)"}
	}
	if p.Destination == "" {
		return &connectors.ValidationError{Message: "missing required parameter: destination"}
	}
	if !validIATACode(p.Destination) {
		return &connectors.ValidationError{Message: "destination must be a 3-letter IATA code (e.g., LAX)"}
	}
	if p.DepartureDate == "" {
		return &connectors.ValidationError{Message: "missing required parameter: departure_date"}
	}
	if !validDate(p.DepartureDate) {
		return &connectors.ValidationError{Message: "departure_date must be YYYY-MM-DD format"}
	}
	if p.ReturnDate != "" && !validDate(p.ReturnDate) {
		return &connectors.ValidationError{Message: "return_date must be YYYY-MM-DD format"}
	}
	if p.Adults < 1 {
		return &connectors.ValidationError{Message: "adults must be at least 1"}
	}
	if p.Adults > 9 {
		return &connectors.ValidationError{Message: "adults must be at most 9"}
	}
	if p.Cabin != "" && !validCabins[p.Cabin] {
		return &connectors.ValidationError{Message: "cabin must be one of: ECONOMY, PREMIUM_ECONOMY, BUSINESS, FIRST"}
	}
	return nil
}

func (a *searchFlightsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchFlightsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	// Default adults to 1 if not set (zero value from JSON).
	if params.Adults == 0 {
		params.Adults = 1
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	maxResults := params.MaxResults
	if maxResults <= 0 {
		maxResults = 10
	}
	if maxResults > maxResultsCap {
		maxResults = maxResultsCap
	}

	q := url.Values{
		"originLocationCode":      {params.Origin},
		"destinationLocationCode": {params.Destination},
		"departureDate":           {params.DepartureDate},
		"adults":                  {strconv.Itoa(params.Adults)},
		"max":                     {strconv.Itoa(maxResults)},
	}
	if params.ReturnDate != "" {
		q.Set("returnDate", params.ReturnDate)
	}
	if params.Cabin != "" {
		q.Set("travelClass", params.Cabin)
	}
	if params.Nonstop {
		q.Set("nonStop", "true")
	}

	path := "/v2/shopping/flight-offers?" + q.Encode()

	var resp struct {
		Data         []json.RawMessage      `json:"data"`
		Dictionaries map[string]interface{} `json:"dictionaries"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
