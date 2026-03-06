package expedia

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getBookingAction implements connectors.Action for expedia.get_booking.
// It retrieves booking details via GET /v3/itineraries/{itinerary_id}?email={email}.
type getBookingAction struct {
	conn *ExpediaConnector
}

// getBookingParams are the parameters parsed from ActionRequest.Parameters.
type getBookingParams struct {
	ItineraryID string `json:"itinerary_id"`
	Email       string `json:"email"`
}

func (p *getBookingParams) validate() error {
	if p.ItineraryID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: itinerary_id"}
	}
	if p.Email == "" {
		return &connectors.ValidationError{Message: "missing required parameter: email"}
	}
	if err := validateEmail(p.Email); err != nil {
		return err
	}
	return nil
}

// Execute retrieves booking details and current status.
func (a *getBookingAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getBookingParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v3/itineraries/%s?email=%s",
		url.PathEscape(params.ItineraryID),
		url.QueryEscape(params.Email),
	)

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, defaultCustomerIP, nil, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
