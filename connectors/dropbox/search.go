package dropbox

import (
	"context"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchAction implements connectors.Action for dropbox.search.
type searchAction struct {
	conn *DropboxConnector
}

type searchParams struct {
	Query          string   `json:"query"`
	Path           string   `json:"path,omitempty"`
	MaxResults     int      `json:"max_results,omitempty"`
	FileExtensions []string `json:"file_extensions,omitempty"`
}

func (p *searchParams) validate() error {
	if p.Query == "" {
		return &connectors.ValidationError{Message: "missing required parameter: query"}
	}
	if p.MaxResults != 0 && (p.MaxResults < 1 || p.MaxResults > 1000) {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("max_results must be between 1 and 1000, got %d", p.MaxResults),
		}
	}
	return nil
}

type searchOptions struct {
	Path           string   `json:"path,omitempty"`
	MaxResults     int      `json:"max_results,omitempty"`
	FileExtensions []string `json:"file_extensions,omitempty"`
}

type searchRequest struct {
	Query   string         `json:"query"`
	Options *searchOptions `json:"options,omitempty"`
}

type searchResponse struct {
	Matches []searchMatch `json:"matches"`
	HasMore bool          `json:"has_more"`
}

type searchMatch struct {
	Metadata searchMatchMetadata `json:"metadata"`
}

type searchMatchMetadata struct {
	Metadata searchFileMetadata `json:"metadata"`
}

type searchFileMetadata struct {
	Tag         string `json:".tag"`
	Name        string `json:"name"`
	PathDisplay string `json:"path_display"`
	ID          string `json:"id"`
	Size        int64  `json:"size,omitempty"`
}

type searchResultItem struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	PathDisplay string `json:"path_display"`
	ID          string `json:"id"`
	Size        int64  `json:"size,omitempty"`
}

func (a *searchAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	body := searchRequest{Query: params.Query}
	if params.Path != "" || params.MaxResults != 0 || len(params.FileExtensions) > 0 {
		opts := &searchOptions{}
		if params.Path != "" {
			opts.Path = params.Path
		}
		if params.MaxResults != 0 {
			opts.MaxResults = params.MaxResults
		}
		if len(params.FileExtensions) > 0 {
			opts.FileExtensions = params.FileExtensions
		}
		body.Options = opts
	}

	var resp searchResponse
	if err := a.conn.doRPC(ctx, "files/search_v2", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	items := make([]searchResultItem, len(resp.Matches))
	for i, m := range resp.Matches {
		items[i] = searchResultItem{
			Type:        m.Metadata.Metadata.Tag,
			Name:        m.Metadata.Metadata.Name,
			PathDisplay: m.Metadata.Metadata.PathDisplay,
			ID:          m.Metadata.Metadata.ID,
			Size:        m.Metadata.Metadata.Size,
		}
	}

	return connectors.JSONResult(map[string]any{
		"matches":  items,
		"has_more": resp.HasMore,
	})
}
