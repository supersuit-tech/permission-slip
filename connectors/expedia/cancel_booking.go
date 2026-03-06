package expedia

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// cancelBookingAction implements connectors.Action for expedia.cancel_booking.
// It cancels a hotel booking via DELETE /v3/itineraries/{itinerary_id}/rooms/{room_id}.
// High risk — may incur cancellation fees.
type cancelBookingAction struct {
	conn *ExpediaConnector
}

// cancelBookingParams are the parameters parsed from ActionRequest.Parameters.
type cancelBookingParams struct {
	ItineraryID string `json:"itinerary_id"`
	RoomID      string `json:"room_id"`
}

func (p *cancelBookingParams) validate() error {
	if p.ItineraryID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: itinerary_id"}
	}
	if p.RoomID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: room_id"}
	}
	return nil
}

// Execute cancels a hotel booking.
func (a *cancelBookingAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params cancelBookingParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v3/itineraries/%s/rooms/%s",
		url.PathEscape(params.ItineraryID),
		url.PathEscape(params.RoomID),
	)

	// Expedia returns 204 No Content on successful cancellation.
	// Pass nil for respBody since there may be no response body.
	if err := a.conn.do(ctx, req.Credentials, http.MethodDelete, path, defaultCustomerIP, nil, nil); err != nil {
		return nil, err
	}

	result := map[string]string{
		"status":       "cancelled",
		"itinerary_id": params.ItineraryID,
		"room_id":      params.RoomID,
	}
	return connectors.JSONResult(result)
}
