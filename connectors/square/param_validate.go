package square

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *SquareConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"square.adjust_inventory": makeParamValidator[adjustInventoryParams](),
	"square.create_booking": makeParamValidator[createBookingParams](),
	"square.create_customer": makeParamValidator[createCustomerParams](),
	"square.create_loyalty_reward": makeParamValidator[createLoyaltyRewardParams](),
	"square.create_order": makeParamValidator[createOrderParams](),
	"square.create_payment": makeParamValidator[createPaymentParams](),
	"square.get_customer": makeParamValidator[getSquareCustomerParams](),
	"square.get_inventory": makeParamValidator[getInventoryParams](),
	"square.issue_refund": makeParamValidator[issueRefundParams](),
	"square.list_customers": makeParamValidator[listSquareCustomersParams](),
	"square.search_orders": makeParamValidator[searchOrdersParams](),
	"square.send_invoice": makeParamValidator[sendInvoiceParams](),
	"square.update_catalog_item": makeParamValidator[updateCatalogItemParams](),
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
