package amadeus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchCarsAction implements connectors.Action for amadeus.search_cars.
// It searches available transfer/rental car offers via POST /v2/shopping/transfer-offers.
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
	if !validIATACode(p.PickupLocation) {
		return &connectors.ValidationError{Message: "pickup_location must be a 3-letter IATA code (e.g., LAX)"}
	}
	if p.DropoffLocation != "" && !validIATACode(p.DropoffLocation) {
		return &connectors.ValidationError{Message: "dropoff_location must be a 3-letter IATA code (e.g., SFO)"}
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

	// Default dropoff to pickup location for round-trip rentals.
	endLocation := params.DropoffLocation
	if endLocation == "" {
		endLocation = params.PickupLocation
	}

	// Build the transfer search request body per Amadeus Transfer API spec.
	body := map[string]any{
		"startLocationCode": params.PickupLocation,
		"endLocationCode":   endLocation,
		"startDateTime":     params.PickupDate,
		"endDateTime":       params.DropoffDate,
		"startConnectedSegment": map[string]string{
			"iataCode": params.PickupLocation,
		},
		"endConnectedSegment": map[string]string{
			"iataCode": endLocation,
		},
	}

	if params.Provider != "" {
		body["transferType"] = params.Provider
	}

	var resp struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/v2/shopping/transfer-offers", body, &resp, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
