package square

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createLoyaltyRewardAction implements connectors.Action for square.create_loyalty_reward.
// It creates a loyalty reward for a customer via POST /v2/loyalty/rewards.
type createLoyaltyRewardAction struct {
	conn *SquareConnector
}

type createLoyaltyRewardParams struct {
	LoyaltyAccountID string `json:"loyalty_account_id"`
	RewardTierID     string `json:"reward_tier_id"`
	OrderID          string `json:"order_id,omitempty"`
}

func (p *createLoyaltyRewardParams) validate() error {
	if p.LoyaltyAccountID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: loyalty_account_id"}
	}
	if p.RewardTierID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: reward_tier_id"}
	}
	return nil
}

// Execute creates a loyalty reward for a customer's loyalty account.
func (a *createLoyaltyRewardAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createLoyaltyRewardParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	reward := map[string]interface{}{
		"loyalty_account_id": params.LoyaltyAccountID,
		"reward_tier_id":     params.RewardTierID,
	}
	if params.OrderID != "" {
		reward["order_id"] = params.OrderID
	}

	reqBody := map[string]interface{}{
		"reward":           reward,
		"idempotency_key":  newIdempotencyKey(),
	}

	var resp struct {
		Reward json.RawMessage `json:"reward"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/loyalty/rewards", reqBody, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
