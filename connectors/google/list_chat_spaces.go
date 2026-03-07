package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listChatSpacesAction implements connectors.Action for google.list_chat_spaces.
// It lists spaces via the Google Chat API GET /v1/spaces.
type listChatSpacesAction struct {
	conn *GoogleConnector
}

// listChatSpacesParams is the user-facing parameter schema.
type listChatSpacesParams struct {
	PageSize int    `json:"page_size"`
	Filter   string `json:"filter"`
}

func (p *listChatSpacesParams) normalize() {
	if p.PageSize <= 0 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
}

// chatSpacesResponse is the Google Chat API response from spaces.list.
type chatSpacesResponse struct {
	Spaces []chatSpace `json:"spaces"`
}

type chatSpace struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Type        string `json:"type"`
	SpaceType   string `json:"spaceType"`
}

// chatSpaceSummary is the shape returned to the agent.
type chatSpaceSummary struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
	SpaceType   string `json:"space_type,omitempty"`
}

// Execute lists Google Chat spaces accessible to the authenticated user.
func (a *listChatSpacesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listChatSpacesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	params.normalize()

	q := url.Values{}
	q.Set("pageSize", strconv.Itoa(params.PageSize))
	if params.Filter != "" {
		q.Set("filter", params.Filter)
	}

	var resp chatSpacesResponse
	listURL := a.conn.chatBaseURL + "/v1/spaces?" + q.Encode()
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, listURL, nil, &resp); err != nil {
		return nil, err
	}

	spaces := make([]chatSpaceSummary, 0, len(resp.Spaces))
	for _, s := range resp.Spaces {
		spaces = append(spaces, chatSpaceSummary{
			Name:        s.Name,
			DisplayName: s.DisplayName,
			Type:        s.Type,
			SpaceType:   s.SpaceType,
		})
	}

	return connectors.JSONResult(map[string]any{
		"spaces": spaces,
	})
}
