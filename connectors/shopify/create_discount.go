package shopify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createDiscountAction implements connectors.Action for shopify.create_discount.
// It creates a discount code via a two-step flow:
//  1. POST /admin/api/2024-10/price_rules.json — creates the price rule
//  2. POST /admin/api/2024-10/price_rules/{id}/discount_codes.json — creates the code
type createDiscountAction struct {
	conn *ShopifyConnector
}

// createDiscountParams maps the JSON parameters for the create_discount action.
// Value must be a negative decimal string (e.g. "-10.0" for 10% off).
type createDiscountParams struct {
	Code                  string `json:"code"`
	ValueType             string `json:"value_type"`
	Value                 string `json:"value"`
	TargetType            string `json:"target_type,omitempty"`
	StartsAt              string `json:"starts_at"`
	EndsAt                string `json:"ends_at,omitempty"`
	UsageLimit            *int   `json:"usage_limit,omitempty"`
	AppliesOncePerCustomer *bool  `json:"applies_once_per_customer,omitempty"`
}

// validValueTypes are the discount value types accepted by the Shopify Price Rules API.
// See: https://shopify.dev/docs/api/admin-rest/2024-10/resources/price-rule#post-price-rules
var validValueTypes = map[string]bool{
	"percentage": true, "fixed_amount": true,
}

// validTargetTypes are the target types for price rules — what the discount applies to.
var validTargetTypes = map[string]bool{
	"line_item": true, "shipping_line": true,
}

func (p *createDiscountParams) validate() error {
	if p.Code == "" {
		return &connectors.ValidationError{Message: "missing required parameter: code"}
	}
	if p.ValueType == "" {
		return &connectors.ValidationError{Message: "missing required parameter: value_type"}
	}
	if !validValueTypes[p.ValueType] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid value_type %q: must be percentage or fixed_amount", p.ValueType)}
	}
	if p.Value == "" {
		return &connectors.ValidationError{Message: "missing required parameter: value"}
	}
	if p.TargetType != "" && !validTargetTypes[p.TargetType] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid target_type %q: must be line_item or shipping_line", p.TargetType)}
	}
	if p.StartsAt == "" {
		return &connectors.ValidationError{Message: "missing required parameter: starts_at"}
	}
	return nil
}

// Execute creates a discount code via a two-step Shopify API flow. If step 2
// fails, the error message includes the orphaned price rule ID for cleanup.
func (a *createDiscountAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createDiscountParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Step 1: Create the price rule.
	targetType := params.TargetType
	if targetType == "" {
		targetType = "line_item"
	}
	// Shopify requires target_selection and allocation_method for price rules.
	priceRule := map[string]interface{}{
		"title":             params.Code,
		"value_type":        params.ValueType,
		"value":             params.Value,
		"target_type":       targetType,
		"target_selection":  "all",
		"allocation_method": "across",
		"customer_selection": "all",
		"starts_at":         params.StartsAt,
	}
	if params.EndsAt != "" {
		priceRule["ends_at"] = params.EndsAt
	}
	if params.UsageLimit != nil {
		priceRule["usage_limit"] = *params.UsageLimit
	}
	if params.AppliesOncePerCustomer != nil {
		priceRule["once_per_customer"] = *params.AppliesOncePerCustomer
	}

	priceRuleBody := map[string]interface{}{
		"price_rule": priceRule,
	}

	var priceRuleResp struct {
		PriceRule struct {
			ID int64 `json:"id"`
		} `json:"price_rule"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/price_rules.json", priceRuleBody, &priceRuleResp); err != nil {
		return nil, err
	}

	// Step 2: Create the discount code for the price rule.
	discountBody := map[string]interface{}{
		"discount_code": map[string]interface{}{
			"code": params.Code,
		},
	}

	var discountResp struct {
		DiscountCode json.RawMessage `json:"discount_code"`
	}
	path := fmt.Sprintf("/price_rules/%d/discount_codes.json", priceRuleResp.PriceRule.ID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, discountBody, &discountResp); err != nil {
		// Wrap the error with context about the orphaned price rule so the user
		// can clean it up if needed.
		return nil, fmt.Errorf("discount code creation failed (price rule %d was created but has no code attached): %w",
			priceRuleResp.PriceRule.ID, err)
	}

	// Return both the price rule ID and the discount code.
	result := map[string]interface{}{
		"price_rule_id": priceRuleResp.PriceRule.ID,
		"discount_code": discountResp.DiscountCode,
	}

	return connectors.JSONResult(result)
}
