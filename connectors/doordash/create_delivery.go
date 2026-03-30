package doordash

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createDeliveryAction implements connectors.Action for doordash.create_delivery.
// It creates a delivery request via POST /drive/v2/deliveries.
// HIGH RISK — dispatches a real Dasher and charges money.
type createDeliveryAction struct {
	conn *DoorDashConnector
}

type deliveryItem struct {
	Name        string `json:"name"`
	Quantity    int    `json:"quantity"`
	Description string `json:"description,omitempty"`
}

type createDeliveryParams struct {
	PickupAddress             string         `json:"pickup_address"`
	PickupPhone               string         `json:"pickup_phone"`
	PickupBusinessName        string         `json:"pickup_business_name,omitempty"`
	PickupInstructions        string         `json:"pickup_instructions,omitempty"`
	DropoffAddress            string         `json:"dropoff_address"`
	DropoffPhone              string         `json:"dropoff_phone"`
	DropoffContactGivenName   string         `json:"dropoff_contact_given_name"`
	DropoffInstructions       string         `json:"dropoff_instructions,omitempty"`
	OrderValue                *int           `json:"order_value,omitempty"`
	Items                     []deliveryItem `json:"items,omitempty"`
}

func (p *createDeliveryParams) validate() error {
	if p.PickupAddress == "" {
		return &connectors.ValidationError{Message: "missing required parameter: pickup_address"}
	}
	if p.PickupPhone == "" {
		return &connectors.ValidationError{Message: "missing required parameter: pickup_phone"}
	}
	if p.DropoffAddress == "" {
		return &connectors.ValidationError{Message: "missing required parameter: dropoff_address"}
	}
	if p.DropoffPhone == "" {
		return &connectors.ValidationError{Message: "missing required parameter: dropoff_phone"}
	}
	if p.DropoffContactGivenName == "" {
		return &connectors.ValidationError{Message: "missing required parameter: dropoff_contact_given_name"}
	}
	if p.OrderValue != nil && *p.OrderValue < 0 {
		return &connectors.ValidationError{Message: "order_value must not be negative"}
	}
	const maxItems = 100
	if len(p.Items) > maxItems {
		return &connectors.ValidationError{Message: fmt.Sprintf("items list too long: %d items, maximum is %d", len(p.Items), maxItems)}
	}
	for i, item := range p.Items {
		if item.Name == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("items[%d]: missing required field: name", i)}
		}
		if item.Quantity <= 0 {
			return &connectors.ValidationError{Message: fmt.Sprintf("items[%d]: quantity must be a positive integer", i)}
		}
	}
	return nil
}

func (a *createDeliveryAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createDeliveryParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"external_delivery_id":        newUUID(),
		"pickup_address":              params.PickupAddress,
		"pickup_phone_number":         params.PickupPhone,
		"dropoff_address":             params.DropoffAddress,
		"dropoff_phone_number":        params.DropoffPhone,
		"dropoff_contact_given_name":  params.DropoffContactGivenName,
	}
	if params.PickupBusinessName != "" {
		body["pickup_business_name"] = params.PickupBusinessName
	}
	if params.PickupInstructions != "" {
		body["pickup_instructions"] = params.PickupInstructions
	}
	if params.DropoffInstructions != "" {
		body["dropoff_instructions"] = params.DropoffInstructions
	}
	if params.OrderValue != nil {
		body["order_value"] = *params.OrderValue
	}
	if len(params.Items) > 0 {
		body["items"] = params.Items
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/drive/v2/deliveries", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
