package linkedin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// deletePostAction implements connectors.Action for linkedin.delete_post.
// It deletes a post via DELETE /rest/posts/{post_urn_encoded}.
type deletePostAction struct {
	conn *LinkedInConnector
}

// deletePostParams is the user-facing parameter schema.
type deletePostParams struct {
	PostURN string `json:"post_urn"`
}

func (p *deletePostParams) validate() error {
	if p.PostURN == "" {
		return &connectors.ValidationError{Message: "missing required parameter: post_urn"}
	}
	if err := validatePostURN(p.PostURN); err != nil {
		return err
	}
	return nil
}

// Execute deletes a LinkedIn post by its URN.
func (a *deletePostAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deletePostParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	encodedURN := url.PathEscape(params.PostURN)
	apiURL := a.conn.restBaseURL + "/posts/" + encodedURN
	if err := a.conn.do(ctx, req.Credentials, http.MethodDelete, apiURL, nil, nil, true); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"status":   "deleted",
		"post_urn": params.PostURN,
	})
}
