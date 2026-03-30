package expedia

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// getHotelAction implements connectors.Action for expedia.get_hotel.
// It retrieves full hotel details via GET /v3/properties/content/{property_id}.
type getHotelAction struct {
	conn *ExpediaConnector
}

// ParameterAliases maps common agent shorthand to the canonical parameter names.
// Agents sometimes send "check_in"/"check_out" instead of "checkin"/"checkout".
func (a *getHotelAction) ParameterAliases() map[string]string {
	return map[string]string{
		"check_in":  "checkin",
		"check_out": "checkout",
	}
}

// getHotelParams are the parameters parsed from ActionRequest.Parameters.
type getHotelParams struct {
	PropertyID string `json:"property_id"`
	Checkin    string `json:"checkin"`
	Checkout   string `json:"checkout"`
	Occupancy  string `json:"occupancy"`
}

func (p *getHotelParams) validate() error {
	if p.PropertyID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: property_id"}
	}
	if p.Checkin != "" {
		if err := validateDate("checkin", p.Checkin); err != nil {
			return err
		}
	}
	if p.Checkout != "" {
		if err := validateDate("checkout", p.Checkout); err != nil {
			return err
		}
	}
	if p.Checkin != "" && p.Checkout != "" {
		if err := validateDateRange(p.Checkin, p.Checkout); err != nil {
			return err
		}
	}
	return nil
}

// Execute retrieves full hotel details including amenities, images, and policies.
func (a *getHotelAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getHotelParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("language", "en-US")
	if params.Checkin != "" {
		q.Set("checkin", params.Checkin)
	}
	if params.Checkout != "" {
		q.Set("checkout", params.Checkout)
	}
	if params.Occupancy != "" {
		q.Set("occupancy", params.Occupancy)
	}

	path := fmt.Sprintf("/v3/properties/content/%s?%s", url.PathEscape(params.PropertyID), q.Encode())

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, defaultCustomerIP, nil, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
