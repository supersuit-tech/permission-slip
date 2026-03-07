package x

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getMeAction implements connectors.Action for x.get_me.
// It fetches the authenticated user's profile via GET /2/users/me.
type getMeAction struct {
	conn *XConnector
}

// Execute fetches the authenticated user's profile.
func (a *getMeAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	// get_me has no required parameters, but we still parse to reject malformed JSON.
	if len(req.Parameters) > 0 {
		var params map[string]any
		if err := json.Unmarshal(req.Parameters, &params); err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
		}
	}

	var xResp struct {
		Data struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			Username      string `json:"username"`
			Description   string `json:"description"`
			PublicMetrics struct {
				FollowersCount int `json:"followers_count"`
				FollowingCount int `json:"following_count"`
				TweetCount     int `json:"tweet_count"`
			} `json:"public_metrics"`
		} `json:"data"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, "/users/me?user.fields=id,name,username,description,public_metrics", nil, &xResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(xResp.Data)
}
