package linkedin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// getProfileAction implements connectors.Action for linkedin.get_profile.
// It retrieves the authenticated user's profile via GET /v2/userinfo.
type getProfileAction struct {
	conn *LinkedInConnector
}

// userinfoResponse is the OpenID Connect userinfo response from LinkedIn.
type userinfoResponse struct {
	Sub            string `json:"sub"`
	Name           string `json:"name"`
	Email          string `json:"email"`
	Picture        string `json:"picture"`
	GivenName      string `json:"given_name"`
	FamilyName     string `json:"family_name"`
	EmailVerified  bool   `json:"email_verified"`
}

// Execute retrieves the authenticated user's profile and returns it.
func (a *getProfileAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	// No parameters needed — this action returns the current user's profile.
	var params map[string]json.RawMessage
	if len(req.Parameters) > 0 {
		if err := json.Unmarshal(req.Parameters, &params); err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
		}
	}

	var resp userinfoResponse
	url := a.conn.v2BaseURL + "/userinfo"
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, url, nil, &resp, false); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"id":             resp.Sub,
		"name":           resp.Name,
		"email":          resp.Email,
		"picture_url":    resp.Picture,
		"given_name":     resp.GivenName,
		"family_name":    resp.FamilyName,
		"email_verified": resp.EmailVerified,
	})
}
