package doordash

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// getQuoteAction implements connectors.Action for doordash.get_quote.
// It gets a delivery fee estimate and ETA via POST /drive/v2/quotes.
type getQuoteAction struct {
	conn *DoorDashConnector
}

type getQuoteParams struct {
	PickupAddress  string `json:"pickup_address"`
	DropoffAddress string `json:"dropoff_address"`
	PickupPhone    string `json:"pickup_phone"`
	DropoffPhone   string `json:"dropoff_phone"`
	OrderValue     *int   `json:"order_value,omitempty"`
}

func (p *getQuoteParams) validate() error {
	if p.PickupAddress == "" {
		return &connectors.ValidationError{Message: "missing required parameter: pickup_address"}
	}
	if p.DropoffAddress == "" {
		return &connectors.ValidationError{Message: "missing required parameter: dropoff_address"}
	}
	if p.PickupPhone == "" {
		return &connectors.ValidationError{Message: "missing required parameter: pickup_phone"}
	}
	if p.DropoffPhone == "" {
		return &connectors.ValidationError{Message: "missing required parameter: dropoff_phone"}
	}
	if p.OrderValue != nil && *p.OrderValue < 0 {
		return &connectors.ValidationError{Message: "order_value must not be negative"}
	}
	return nil
}

func (a *getQuoteAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getQuoteParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"external_delivery_id":   newUUID(),
		"pickup_address":         params.PickupAddress,
		"dropoff_address":        params.DropoffAddress,
		"pickup_phone_number":    params.PickupPhone,
		"dropoff_phone_number":   params.DropoffPhone,
	}
	if params.OrderValue != nil {
		body["order_value"] = *params.OrderValue
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/drive/v2/quotes", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
