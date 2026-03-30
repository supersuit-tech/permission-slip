package stripe

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *StripeConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"stripe.cancel_subscription": makeParamValidator[cancelSubscriptionParams](),
	"stripe.create_checkout_session": makeParamValidator[createCheckoutSessionParams](),
	"stripe.create_coupon": makeParamValidator[createCouponParams](),
	"stripe.create_customer": makeParamValidator[createCustomerParams](),
	"stripe.create_invoice": makeParamValidator[createInvoiceParams](),
	"stripe.create_payment_link": makeParamValidator[createPaymentLinkParams](),
	"stripe.create_price": makeParamValidator[createPriceParams](),
	"stripe.create_product": makeParamValidator[createProductParams](),
	"stripe.create_promotion_code": makeParamValidator[createPromotionCodeParams](),
	"stripe.create_subscription": makeParamValidator[createSubscriptionParams](),
	"stripe.get_customer": makeParamValidator[getCustomerParams](),
	"stripe.initiate_payout": makeParamValidator[initiatePayoutParams](),
	"stripe.issue_refund": makeParamValidator[issueRefundParams](),
	"stripe.list_charges": makeParamValidator[listChargesParams](),
	"stripe.list_customers": makeParamValidator[listCustomersParams](),
	"stripe.list_invoices": makeParamValidator[listInvoicesParams](),
	"stripe.list_subscriptions": makeParamValidator[listSubscriptionsParams](),
	"stripe.update_subscription": makeParamValidator[updateSubscriptionParams](),
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
