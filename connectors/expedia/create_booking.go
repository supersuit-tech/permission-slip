package expedia

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createBookingAction implements connectors.Action for expedia.create_booking.
// It books a hotel room via POST /v3/itineraries. High risk — creates a real
// reservation and may charge payment.
type createBookingAction struct {
	conn *ExpediaConnector
}

// createBookingParams are the parameters parsed from ActionRequest.Parameters.
type createBookingParams struct {
	RoomID          string `json:"room_id"`
	GivenName       string `json:"given_name"`
	FamilyName      string `json:"family_name"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	PaymentMethodID string `json:"payment_method_id"`
	SpecialRequest  string `json:"special_request"`
}

func (p *createBookingParams) validate() error {
	if p.RoomID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: room_id"}
	}
	if p.GivenName == "" {
		return &connectors.ValidationError{Message: "missing required parameter: given_name"}
	}
	if p.FamilyName == "" {
		return &connectors.ValidationError{Message: "missing required parameter: family_name"}
	}
	if p.Email == "" {
		return &connectors.ValidationError{Message: "missing required parameter: email"}
	}
	if p.Phone == "" {
		return &connectors.ValidationError{Message: "missing required parameter: phone"}
	}
	if p.PaymentMethodID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: payment_method_id"}
	}
	return nil
}

// Execute creates a hotel booking with the Expedia Rapid API.
func (a *createBookingAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createBookingParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Build the booking request body per the Expedia Rapid API format.
	// Payment details are resolved server-side from payment_method_id —
	// the agent never sees raw card data.
	reqBody := map[string]any{
		"room_id": params.RoomID,
		"contact": map[string]string{
			"given_name":  params.GivenName,
			"family_name": params.FamilyName,
			"email":       params.Email,
			"phone":       params.Phone,
		},
		"payment_method_id": params.PaymentMethodID,
	}
	if params.SpecialRequest != "" {
		reqBody["special_request"] = params.SpecialRequest
	}

	path := fmt.Sprintf("/v3/itineraries?room_id=%s", url.QueryEscape(params.RoomID))

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, defaultCustomerIP, reqBody, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
