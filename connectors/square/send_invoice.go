package square

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// sendInvoiceAction implements connectors.Action for square.send_invoice.
// It creates and publishes an invoice as a single atomic operation via
// POST /v2/invoices (create) then POST /v2/invoices/{id}/publish.
// High risk — sends a real payment request to a customer.
type sendInvoiceAction struct {
	conn *SquareConnector
}

type invoiceLineItem struct {
	Description string `json:"description"`
	Quantity    string `json:"quantity"`
	BasePriceMoney money `json:"base_price_money"`
}

type sendInvoiceParams struct {
	CustomerID     string            `json:"customer_id"`
	LocationID     string            `json:"location_id"`
	LineItems      []invoiceLineItem `json:"line_items"`
	DueDate        string            `json:"due_date"`
	DeliveryMethod string            `json:"delivery_method,omitempty"`
	Title          string            `json:"title,omitempty"`
	Note           string            `json:"note,omitempty"`
}

func (p *sendInvoiceParams) validate() error {
	if p.CustomerID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: customer_id"}
	}
	if p.LocationID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: location_id"}
	}
	if len(p.LineItems) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: line_items (must be a non-empty array)"}
	}
	if len(p.LineItems) > 500 {
		return &connectors.ValidationError{Message: "line_items exceeds maximum of 500 items"}
	}
	if p.DueDate == "" {
		return &connectors.ValidationError{Message: "missing required parameter: due_date"}
	}
	if _, err := time.Parse("2006-01-02", p.DueDate); err != nil {
		return &connectors.ValidationError{Message: "due_date must be in YYYY-MM-DD format (e.g. \"2024-12-31\")"}
	}
	for i, item := range p.LineItems {
		if item.Description == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("line_items[%d].description is required", i)}
		}
		if item.Quantity == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("line_items[%d].quantity is required", i)}
		}
		if qty, err := strconv.Atoi(item.Quantity); err != nil || qty <= 0 {
			return &connectors.ValidationError{Message: fmt.Sprintf("line_items[%d].quantity must be a positive integer string (e.g. \"1\")", i)}
		}
		if item.BasePriceMoney.Amount <= 0 {
			return &connectors.ValidationError{Message: fmt.Sprintf("line_items[%d].base_price_money.amount must be greater than 0", i)}
		}
		if item.BasePriceMoney.Currency == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("line_items[%d].base_price_money.currency is required", i)}
		}
	}

	if p.DeliveryMethod != "" {
		validMethods := map[string]bool{
			"EMAIL":       true,
			"SHARE_MANUALLY": true,
			"SMS":         true,
		}
		if !validMethods[p.DeliveryMethod] {
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid delivery_method %q: must be EMAIL, SHARE_MANUALLY, or SMS", p.DeliveryMethod)}
		}
	}
	return nil
}

func (a *sendInvoiceAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendInvoiceParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Derive a stable base idempotency key from the request parameters so
	// retries of the same send_invoice call reuse the same keys across all
	// three steps, preventing duplicate orders/invoices on partial failures.
	baseKey := deriveBaseKey(req.ActionType, req.Parameters)

	// Build line items for the order (invoices require an order).
	orderLineItems := make([]map[string]interface{}, len(params.LineItems))
	for i, item := range params.LineItems {
		orderLineItems[i] = map[string]interface{}{
			"name":             item.Description,
			"quantity":         item.Quantity,
			"base_price_money": item.BasePriceMoney,
		}
	}

	// Step 1: Create an order for the invoice line items.
	orderBody := map[string]interface{}{
		"idempotency_key": baseKey + "-order",
		"order": map[string]interface{}{
			"location_id": params.LocationID,
			"customer_id": params.CustomerID,
			"line_items":  orderLineItems,
		},
	}

	var orderResp struct {
		Order struct {
			ID string `json:"id"`
		} `json:"order"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/orders", orderBody, &orderResp); err != nil {
		return nil, fmt.Errorf("creating order for invoice: %w", err)
	}

	// Step 2: Create the invoice linked to the order.
	deliveryMethod := params.DeliveryMethod
	if deliveryMethod == "" {
		deliveryMethod = "EMAIL"
	}

	paymentRequests := []map[string]interface{}{
		{
			"request_type":          "BALANCE",
			"due_date":              params.DueDate,
			"automatic_payment_source": "NONE",
		},
	}

	invoice := map[string]interface{}{
		"location_id":      params.LocationID,
		"order_id":         orderResp.Order.ID,
		"delivery_method":  deliveryMethod,
		"payment_requests": paymentRequests,
		"primary_recipient": map[string]interface{}{
			"customer_id": params.CustomerID,
		},
	}
	if params.Title != "" {
		invoice["title"] = params.Title
	}
	if params.Note != "" {
		invoice["description"] = params.Note
	}

	createBody := map[string]interface{}{
		"idempotency_key": baseKey + "-invoice",
		"invoice":         invoice,
	}

	var createResp struct {
		Invoice struct {
			ID      string `json:"id"`
			Version int64  `json:"version"`
		} `json:"invoice"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/invoices", createBody, &createResp); err != nil {
		return nil, fmt.Errorf("creating invoice failed (orphaned order %s may need manual cleanup): %w", orderResp.Order.ID, err)
	}

	// Step 3: Publish the invoice to send it to the customer.
	publishBody := map[string]interface{}{
		"idempotency_key": baseKey + "-publish",
		"version":         createResp.Invoice.Version,
	}

	var publishResp struct {
		Invoice json.RawMessage `json:"invoice"`
	}

	publishPath := fmt.Sprintf("/invoices/%s/publish", createResp.Invoice.ID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, publishPath, publishBody, &publishResp); err != nil {
		return nil, fmt.Errorf("publishing invoice %s failed (draft invoice and order %s may need manual cleanup): %w", createResp.Invoice.ID, orderResp.Order.ID, err)
	}

	return connectors.JSONResult(json.RawMessage(publishResp.Invoice))
}
