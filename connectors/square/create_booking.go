package square

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createBookingAction implements connectors.Action for square.create_booking.
// It creates an appointment booking via POST /v2/bookings.
type createBookingAction struct {
	conn *SquareConnector
}

type createBookingParams struct {
	LocationID         string `json:"location_id"`
	CustomerID         string `json:"customer_id,omitempty"`
	StartAt            string `json:"start_at"`
	ServiceVariationID string `json:"service_variation_id"`
	TeamMemberID       string `json:"team_member_id,omitempty"`
	CustomerNote       string `json:"customer_note,omitempty"`
}

func (p *createBookingParams) validate() error {
	if p.LocationID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: location_id"}
	}
	if p.StartAt == "" {
		return &connectors.ValidationError{Message: "missing required parameter: start_at"}
	}
	if _, err := time.Parse(time.RFC3339, p.StartAt); err != nil {
		return &connectors.ValidationError{Message: "start_at must be RFC 3339 format (e.g. \"2024-03-15T14:30:00Z\")"}
	}
	if p.ServiceVariationID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: service_variation_id"}
	}
	return nil
}

func (a *createBookingAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createBookingParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	segment := map[string]interface{}{
		"service_variation_id": params.ServiceVariationID,
	}
	if params.TeamMemberID != "" {
		segment["team_member_id"] = params.TeamMemberID
	}

	booking := map[string]interface{}{
		"location_id":          params.LocationID,
		"start_at":             params.StartAt,
		"appointment_segments": []interface{}{segment},
	}
	if params.CustomerID != "" {
		booking["customer_id"] = params.CustomerID
	}
	if params.CustomerNote != "" {
		booking["customer_note"] = params.CustomerNote
	}

	body := map[string]interface{}{
		"idempotency_key": newIdempotencyKey(),
		"booking":         booking,
	}

	var resp struct {
		Booking json.RawMessage `json:"booking"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/bookings", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(json.RawMessage(resp.Booking))
}
