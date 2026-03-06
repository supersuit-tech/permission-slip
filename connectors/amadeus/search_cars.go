package amadeus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchCarsAction implements connectors.Action for amadeus.search_cars.
// It searches available transfer/rental car offers via GET /v2/shopping/transfer-offers.
type searchCarsAction struct {
	conn *AmadeusConnector
}

type searchCarsParams struct {
	PickupLocation  string `json:"pickup_location"`
	PickupDate      string `json:"pickup_date"`
	DropoffDate     string `json:"dropoff_date"`
	DropoffLocation string `json:"dropoff_location"`
	Provider        string `json:"provider"`
}

func (p *searchCarsParams) validate() error {
	if p.PickupLocation == "" {
		return &connectors.ValidationError{Message: "missing required parameter: pickup_location"}
	}
	if p.PickupDate == "" {
		return &connectors.ValidationError{Message: "missing required parameter: pickup_date"}
	}
	if p.DropoffDate == "" {
		return &connectors.ValidationError{Message: "missing required parameter: dropoff_date"}
	}
	return nil
}

func (a *searchCarsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchCarsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Build the transfer search request body per Amadeus Transfer API spec.
	startLocation := map[string]string{"iataCode": params.PickupLocation}
	endLocation := startLocation
	if params.DropoffLocation != "" {
		endLocation = map[string]string{"iataCode": params.DropoffLocation}
	}

	body := map[string]any{
		"startLocationCode":    params.PickupLocation,
		"endLocationCode":      params.DropoffLocation,
		"startDateTime":        params.PickupDate,
		"endDateTime":          params.DropoffDate,
		"startConnectedSegment": startLocation,
		"endConnectedSegment":   endLocation,
	}

	// Use query params approach for transfer offers search.
	q := url.Values{
		"startLocationCode": {params.PickupLocation},
		"startDateTime":     {params.PickupDate},
		"endDateTime":       {params.DropoffDate},
	}
	if params.DropoffLocation != "" {
		q.Set("endLocationCode", params.DropoffLocation)
	} else {
		q.Set("endLocationCode", params.PickupLocation)
	}
	if params.Provider != "" {
		q.Set("transferType", params.Provider)
	}

	// The Amadeus Transfer API uses POST for search with a body,
	// unlike the flight search which uses GET with query params.
	path := "/v2/shopping/transfer-offers"

	var resp struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
