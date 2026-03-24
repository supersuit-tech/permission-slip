package paypal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type createInvoiceAction struct {
	conn *PayPalConnector
}

type createInvoiceParams struct {
	Invoice json.RawMessage `json:"invoice"`
}

func (a *createInvoiceAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createInvoiceParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	body, err := readJSONBody(params.Invoice, "invoice")
	if err != nil {
		return nil, err
	}
	reqID := deriveRequestID(req.ActionType, req.Parameters)
	raw, err := a.conn.doJSONRaw(ctx, req.Credentials, http.MethodPost, "/v2/invoicing/invoices", body, reqID)
	if err != nil {
		return nil, err
	}
	return connectors.JSONResult(raw)
}
