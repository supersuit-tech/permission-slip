package square

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createOrderAction implements connectors.Action for square.create_order.
// It creates a new order at a Square location via POST /v2/orders.
type createOrderAction struct {
	conn *SquareConnector
}

type createOrderParams struct {
	LocationID string              `json:"location_id"`
	LineItems  []orderLineItem     `json:"line_items"`
	CustomerID string              `json:"customer_id,omitempty"`
	Note       string              `json:"note,omitempty"`
}

type orderLineItem struct {
	Name           string `json:"name"`
	Quantity       string `json:"quantity"`
	BasePriceMoney money  `json:"base_price_money"`
}

type money struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

func (p *createOrderParams) validate() error {
	if p.LocationID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: location_id"}
	}
	if len(p.LineItems) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: line_items (must have at least one item)"}
	}
	for i, item := range p.LineItems {
		if item.Name == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("line_items[%d]: missing required field: name", i)}
		}
		if item.Quantity == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("line_items[%d]: missing required field: quantity", i)}
		}
		if qty, err := strconv.Atoi(item.Quantity); err != nil || qty <= 0 {
			return &connectors.ValidationError{Message: fmt.Sprintf("line_items[%d]: quantity must be a positive integer string (e.g. \"1\")", i)}
		}
		if item.BasePriceMoney.Currency == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("line_items[%d]: missing required field: base_price_money.currency", i)}
		}
		if item.BasePriceMoney.Amount < 0 {
			return &connectors.ValidationError{Message: fmt.Sprintf("line_items[%d]: base_price_money.amount must not be negative", i)}
		}
	}
	return nil
}

func (a *createOrderAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createOrderParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	order := map[string]interface{}{
		"location_id": params.LocationID,
		"line_items":  params.LineItems,
	}
	if params.CustomerID != "" {
		order["customer_id"] = params.CustomerID
	}
	if params.Note != "" {
		order["note"] = params.Note
	}

	body := map[string]interface{}{
		"idempotency_key": newIdempotencyKey(),
		"order":           order,
	}

	var resp struct {
		Order json.RawMessage `json:"order"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/orders", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(json.RawMessage(resp.Order))
}
