package quickbooks

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *QuickBooksConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"quickbooks.create_bill": makeParamValidator[createBillParams](),
	"quickbooks.create_customer": makeParamValidator[createCustomerParams](),
	"quickbooks.create_expense": makeParamValidator[createExpenseParams](),
	"quickbooks.create_invoice": makeParamValidator[createInvoiceParams](),
	"quickbooks.create_vendor": makeParamValidator[createVendorParams](),
	"quickbooks.list_invoices": makeParamValidator[listInvoicesParams](),
	"quickbooks.reconcile_transaction": makeParamValidator[reconcileTransactionParams](),
	"quickbooks.record_payment": makeParamValidator[recordPaymentParams](),
	"quickbooks.send_invoice": makeParamValidator[sendInvoiceParams](),
}

func makeParamValidator[T any, PT interface {
	*T
	validate() error
}]() connectors.ParamValidatorFunc {
	return func(params json.RawMessage) error {
		p := PT(new(T))
		if err := json.Unmarshal(params, p); err != nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
		}
		return p.validate()
	}
}
