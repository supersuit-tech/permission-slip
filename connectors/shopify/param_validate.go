package shopify

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *ShopifyConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"shopify.cancel_order": makeParamValidator[cancelOrderParams](),
	"shopify.create_collection": makeParamValidator[createCollectionParams](),
	"shopify.create_customer": makeParamValidator[createCustomerParams](),
	"shopify.create_discount": makeParamValidator[createDiscountParams](),
	"shopify.create_draft_order": makeParamValidator[createDraftOrderParams](),
	"shopify.create_product": makeParamValidator[createProductParams](),
	"shopify.fulfill_order": makeParamValidator[fulfillOrderParams](),
	"shopify.get_analytics": makeParamValidator[getAnalyticsParams](),
	"shopify.get_customer": makeParamValidator[getCustomerParams](),
	"shopify.get_order": makeParamValidator[getOrderParams](),
	"shopify.get_orders": makeParamValidator[getOrdersParams](),
	"shopify.get_product": makeParamValidator[getProductParams](),
	"shopify.list_customers": makeParamValidator[listCustomersParams](),
	"shopify.list_products": makeParamValidator[listProductsParams](),
	"shopify.update_inventory": makeParamValidator[updateInventoryParams](),
	"shopify.update_order": makeParamValidator[updateOrderParams](),
	"shopify.update_product": makeParamValidator[updateProductParams](),
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
