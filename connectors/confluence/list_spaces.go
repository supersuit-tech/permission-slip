package confluence

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listSpacesAction implements connectors.Action for confluence.list_spaces.
// It lists available spaces via GET /wiki/api/v2/spaces.
type listSpacesAction struct {
	conn *ConfluenceConnector
}

type listSpacesParams struct {
	Limit  int    `json:"limit,omitempty"`
	Status string `json:"status,omitempty"`
}

func (p *listSpacesParams) validate() error {
	if p.Limit < 0 || p.Limit > 250 {
		return &connectors.ValidationError{Message: "limit must be between 1 and 250"}
	}
	validStatuses := map[string]bool{"current": true, "archived": true, "": true}
	if !validStatuses[strings.ToLower(p.Status)] {
		return &connectors.ValidationError{Message: "status must be one of: current, archived"}
	}
	return nil
}

type listSpacesResponse struct {
	Results []spaceItem `json:"results"`
	Links   struct {
		Next string `json:"next,omitempty"`
	} `json:"_links"`
}

type spaceItem struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Description struct {
		Plain struct {
			Value string `json:"value"`
		} `json:"plain"`
	} `json:"description"`
	Links struct {
		WebUI string `json:"webui"`
	} `json:"_links"`
}

func (a *listSpacesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listSpacesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit == 0 {
		limit = 25
	}

	q := url.Values{}
	q.Set("limit", fmt.Sprintf("%d", limit))
	if params.Status != "" {
		q.Set("status", strings.ToLower(params.Status))
	}

	path := "/spaces?" + q.Encode()

	var resp listSpacesResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
