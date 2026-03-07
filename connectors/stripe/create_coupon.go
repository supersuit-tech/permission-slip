package stripe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createCouponAction implements connectors.Action for stripe.create_coupon.
// It creates a discount coupon via POST /v1/coupons.
type createCouponAction struct {
	conn *StripeConnector
}

type createCouponParams struct {
	PercentOff     *float64       `json:"percent_off"`
	AmountOff      *int64         `json:"amount_off"`
	Currency       string         `json:"currency"`
	Duration       string         `json:"duration"`
	DurationMonths *int64         `json:"duration_in_months"`
	MaxRedemptions *int64         `json:"max_redemptions"`
	Name           string         `json:"name"`
	Metadata       map[string]any `json:"metadata"`
}

func (p *createCouponParams) validate() error {
	if p.PercentOff == nil && p.AmountOff == nil {
		return &connectors.ValidationError{Message: "either percent_off or amount_off is required"}
	}
	if p.PercentOff != nil && p.AmountOff != nil {
		return &connectors.ValidationError{Message: "provide either percent_off or amount_off, not both"}
	}
	if p.PercentOff != nil && (*p.PercentOff <= 0 || *p.PercentOff > 100) {
		return &connectors.ValidationError{Message: "percent_off must be between 0 (exclusive) and 100 (inclusive)"}
	}
	if p.AmountOff != nil {
		if *p.AmountOff <= 0 {
			return &connectors.ValidationError{Message: "amount_off must be positive (in smallest currency unit)"}
		}
		if p.Currency == "" {
			return &connectors.ValidationError{Message: "currency is required when using amount_off"}
		}
	}

	validDurations := map[string]bool{
		"once":      true,
		"repeating": true,
		"forever":   true,
	}
	if p.Duration == "" {
		return &connectors.ValidationError{Message: "missing required parameter: duration"}
	}
	if !validDurations[p.Duration] {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid duration %q: must be one of once, repeating, forever", p.Duration),
		}
	}
	if p.Duration == "repeating" && (p.DurationMonths == nil || *p.DurationMonths <= 0) {
		return &connectors.ValidationError{Message: "duration_in_months is required and must be positive when duration is \"repeating\""}
	}
	if p.Duration != "repeating" && p.DurationMonths != nil {
		return &connectors.ValidationError{Message: "duration_in_months is only valid when duration is \"repeating\""}
	}
	if p.MaxRedemptions != nil && *p.MaxRedemptions <= 0 {
		return &connectors.ValidationError{Message: "max_redemptions must be positive"}
	}
	return validateMetadata(p.Metadata)
}

// Execute creates a Stripe coupon and returns the coupon data.
func (a *createCouponAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createCouponParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"duration": params.Duration,
	}
	if params.PercentOff != nil {
		body["percent_off"] = *params.PercentOff
	}
	if params.AmountOff != nil {
		body["amount_off"] = *params.AmountOff
		body["currency"] = params.Currency
	}
	if params.DurationMonths != nil {
		body["duration_in_months"] = *params.DurationMonths
	}
	if params.MaxRedemptions != nil {
		body["max_redemptions"] = *params.MaxRedemptions
	}
	if params.Name != "" {
		body["name"] = params.Name
	}
	if params.Metadata != nil {
		body["metadata"] = params.Metadata
	}

	formParams := formEncode(body)

	var resp struct {
		ID               string  `json:"id"`
		Name             string  `json:"name"`
		PercentOff       float64 `json:"percent_off"`
		AmountOff        int64   `json:"amount_off"`
		Currency         string  `json:"currency"`
		Duration         string  `json:"duration"`
		DurationInMonths *int64  `json:"duration_in_months"`
		MaxRedemptions   *int64  `json:"max_redemptions"`
		TimesRedeemed    int64   `json:"times_redeemed"`
		Valid            bool    `json:"valid"`
		Created          int64   `json:"created"`
	}

	if err := a.conn.doPost(ctx, req.Credentials, "/v1/coupons", formParams, &resp, req.ActionType, req.Parameters); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
