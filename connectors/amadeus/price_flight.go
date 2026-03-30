package amadeus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// priceFlightAction implements connectors.Action for amadeus.price_flight.
// It confirms real-time pricing for a flight offer via POST /v1/shopping/flight-offers/pricing.
type priceFlightAction struct {
	conn *AmadeusConnector
}

type priceFlightParams struct {
	FlightOffer json.RawMessage `json:"flight_offer"`
}

func (p *priceFlightParams) validate() error {
	if len(p.FlightOffer) == 0 || string(p.FlightOffer) == "null" {
		return &connectors.ValidationError{Message: "missing required parameter: flight_offer"}
	}
	if len(p.FlightOffer) > maxFlightOfferBytes {
		return &connectors.ValidationError{Message: "flight_offer exceeds maximum size"}
	}
	return nil
}

func (a *priceFlightAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params priceFlightParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Build the pricing request body per Amadeus spec.
	body := map[string]any{
		"data": map[string]any{
			"type":         "flight-offers-pricing",
			"flightOffers": []json.RawMessage{params.FlightOffer},
		},
	}

	var resp struct {
		Data json.RawMessage `json:"data"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/v1/shopping/flight-offers/pricing", body, &resp, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
