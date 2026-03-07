package sendgrid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createTemplateAction implements connectors.Action for sendgrid.create_template.
// It creates a dynamic transactional email template via POST /templates.
type createTemplateAction struct {
	conn *SendGridConnector
}

type createTemplateParams struct {
	Name       string `json:"name"`
	Generation string `json:"generation"`
}

func (p *createTemplateParams) validate() error {
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	if len(p.Name) > 100 {
		return &connectors.ValidationError{Message: fmt.Sprintf("name exceeds maximum length of 100 characters (got %d)", len(p.Name))}
	}
	if p.Generation != "" && p.Generation != "legacy" && p.Generation != "dynamic" {
		return &connectors.ValidationError{Message: fmt.Sprintf("generation must be \"legacy\" or \"dynamic\", got %q", p.Generation)}
	}
	return nil
}

// Execute creates a new email template.
func (a *createTemplateAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createTemplateParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	generation := params.Generation
	if generation == "" {
		generation = "dynamic"
	}

	body := map[string]string{
		"name":       params.Name,
		"generation": generation,
	}

	var resp struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Generation string `json:"generation"`
		UpdatedAt  string `json:"updated_at"`
	}
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, "/templates", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"template_id": resp.ID,
		"name":        resp.Name,
		"generation":  resp.Generation,
	})
}
