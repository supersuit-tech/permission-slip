package stripe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createPromotionCodeAction implements connectors.Action for stripe.create_promotion_code.
// It creates a shareable promotion code for an existing coupon via POST /v1/promotion_codes.
type createPromotionCodeAction struct {
	conn *StripeConnector
}

type createPromotionCodeParams struct {
	Coupon         string         `json:"coupon"`
	Code           string         `json:"code"`
	MaxRedemptions *int64         `json:"max_redemptions"`
	ExpiresAt      *int64         `json:"expires_at"`
	Metadata       map[string]any `json:"metadata"`
}

func (p *createPromotionCodeParams) validate() error {
	if p.Coupon == "" {
		return &connectors.ValidationError{Message: "missing required parameter: coupon"}
	}
	if p.MaxRedemptions != nil && *p.MaxRedemptions <= 0 {
		return &connectors.ValidationError{Message: "max_redemptions must be positive"}
	}
	if p.ExpiresAt != nil && *p.ExpiresAt <= 0 {
		return &connectors.ValidationError{Message: "expires_at must be a positive Unix timestamp"}
	}
	return validateMetadata(p.Metadata)
}

// Execute creates a Stripe promotion code and returns the promotion code data.
func (a *createPromotionCodeAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createPromotionCodeParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"coupon": params.Coupon,
	}
	if params.Code != "" {
		body["code"] = params.Code
	}
	if params.MaxRedemptions != nil {
		body["max_redemptions"] = *params.MaxRedemptions
	}
	if params.ExpiresAt != nil {
		body["expires_at"] = *params.ExpiresAt
	}
	if params.Metadata != nil {
		body["metadata"] = params.Metadata
	}

	formParams := formEncode(body)

	var resp struct {
		ID             string `json:"id"`
		Code           string `json:"code"`
		Coupon         string `json:"coupon"`
		Active         bool   `json:"active"`
		MaxRedemptions *int64 `json:"max_redemptions"`
		ExpiresAt      *int64 `json:"expires_at"`
	}

	if err := a.conn.doPost(ctx, req.Credentials, "/v1/promotion_codes", formParams, &resp, req.ActionType, req.Parameters); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
