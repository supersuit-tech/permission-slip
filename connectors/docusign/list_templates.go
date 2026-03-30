package docusign

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listTemplatesAction implements connectors.Action for docusign.list_templates.
// It lists available templates via GET /templates.
type listTemplatesAction struct {
	conn *DocuSignConnector
}

type listTemplatesParams struct {
	SearchText    string `json:"search_text"`
	Count         int    `json:"count"`
	StartPosition int    `json:"start_position"`
}

func (p *listTemplatesParams) validate() error {
	if p.Count != 0 && (p.Count < 1 || p.Count > 100) {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("count must be between 1 and 100, got %d", p.Count),
		}
	}
	if p.StartPosition < 0 {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("start_position must be non-negative, got %d", p.StartPosition),
		}
	}
	return nil
}

type listTemplatesResponse struct {
	EnvelopeTemplates []templateSummary `json:"envelopeTemplates"`
	ResultSetSize     string            `json:"resultSetSize"`
	TotalSetSize      string            `json:"totalSetSize"`
	StartPosition     string            `json:"startPosition"`
}

type templateSummary struct {
	TemplateID   string `json:"templateId"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Created      string `json:"created"`
	LastModified string `json:"lastModified"`
	Shared       string `json:"shared"`
	FolderName   string `json:"folderName"`
}

func (a *listTemplatesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listTemplatesParams
	accountID, err := parseParams(req, &params)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	if params.SearchText != "" {
		query.Set("search_text", params.SearchText)
	}
	if params.Count > 0 {
		query.Set("count", strconv.Itoa(params.Count))
	}
	if params.StartPosition > 0 {
		query.Set("start_position", strconv.Itoa(params.StartPosition))
	}

	path := accountPath(accountID) + "/templates"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var resp listTemplatesResponse
	if err := a.conn.doJSON(ctx, "GET", path, req.Credentials, nil, &resp); err != nil {
		return nil, err
	}

	type templateResult struct {
		TemplateID   string `json:"template_id"`
		Name         string `json:"name"`
		Description  string `json:"description"`
		Created      string `json:"created,omitempty"`
		LastModified string `json:"last_modified,omitempty"`
		FolderName   string `json:"folder_name,omitempty"`
	}

	templates := make([]templateResult, len(resp.EnvelopeTemplates))
	for i, t := range resp.EnvelopeTemplates {
		templates[i] = templateResult{
			TemplateID:   t.TemplateID,
			Name:         t.Name,
			Description:  t.Description,
			Created:      t.Created,
			LastModified: t.LastModified,
			FolderName:   t.FolderName,
		}
	}

	return connectors.JSONResult(map[string]any{
		"templates":       templates,
		"result_set_size": resp.ResultSetSize,
		"total_set_size":  resp.TotalSetSize,
		"start_position":  resp.StartPosition,
	})
}
