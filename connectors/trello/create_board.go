package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type createBoardAction struct {
	conn *TrelloConnector
}

type createBoardParams struct {
	Name                  string `json:"name"`
	Desc                  string `json:"desc"`
	DefaultLists          *bool  `json:"defaultLists"`
	OrganizationID        string `json:"idOrganization"`
	PermissionLevel       string `json:"prefs_permissionLevel"`
}

func (p *createBoardParams) validate() error {
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	return nil
}

func (a *createBoardAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createBoardParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"name": params.Name,
	}
	if params.Desc != "" {
		body["desc"] = params.Desc
	}
	if params.DefaultLists != nil {
		body["defaultLists"] = *params.DefaultLists
	}
	if params.OrganizationID != "" {
		body["idOrganization"] = params.OrganizationID
	}
	if params.PermissionLevel != "" {
		body["prefs_permissionLevel"] = params.PermissionLevel
	}

	var resp struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Desc     string `json:"desc"`
		ShortURL string `json:"shortUrl"`
		URL      string `json:"url"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/boards", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
