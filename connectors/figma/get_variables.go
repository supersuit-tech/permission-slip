package figma

import (
	"context"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getVariablesAction implements connectors.Action for figma.get_variables.
// It retrieves local design system variables from a file via GET /files/{file_key}/variables/local.
// Variables are a newer Figma feature for design tokens (colors, spacing, typography, etc.)
// that can be bound to component properties.
type getVariablesAction struct {
	conn *FigmaConnector
}

type getVariablesParams struct {
	FileKey string `json:"file_key"`
}

func (p *getVariablesParams) validate() error {
	p.FileKey = extractFileKey(p.FileKey)
	return validateFileKey(p.FileKey)
}

// figmaVariablesResponse is the response from GET /files/{key}/variables/local.
// It contains variable collections (groups) and individual variables.
type figmaVariablesResponse struct {
	Status    int                             `json:"status"`
	Error     bool                            `json:"error"`
	Meta      figmaVariablesMeta              `json:"meta"`
}

type figmaVariablesMeta struct {
	Variables            map[string]figmaVariable           `json:"variables"`
	VariableCollections  map[string]figmaVariableCollection `json:"variableCollections"`
}

type figmaVariable struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Key              string                 `json:"key"`
	VariableCollectionID string             `json:"variableCollectionId"`
	ResolvedType     string                 `json:"resolvedType"`
	ValuesByMode     map[string]interface{} `json:"valuesByMode"`
	Remote           bool                   `json:"remote"`
	Description      string                 `json:"description,omitempty"`
	HiddenFromPublishing bool               `json:"hiddenFromPublishing"`
	Scopes           []string               `json:"scopes"`
}

type figmaVariableCollection struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Key            string   `json:"key"`
	Modes          []figmaVariableMode `json:"modes"`
	DefaultModeID  string   `json:"defaultModeId"`
	Remote         bool     `json:"remote"`
	HiddenFromPublishing bool `json:"hiddenFromPublishing"`
}

type figmaVariableMode struct {
	ModeID string `json:"modeId"`
	Name   string `json:"name"`
}

func (a *getVariablesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getVariablesParams
	if err := parseParams(req.Parameters, &params); err != nil {
		return nil, err
	}

	var resp figmaVariablesResponse
	path := fmt.Sprintf("/files/%s/variables/local", url.PathEscape(params.FileKey))
	if err := a.conn.doGet(ctx, path, req.Credentials, &resp); err != nil {
		return nil, err
	}

	// Return summary with counts alongside the full data.
	return connectors.JSONResult(map[string]interface{}{
		"variables":            resp.Meta.Variables,
		"variable_collections": resp.Meta.VariableCollections,
		"variable_count":       len(resp.Meta.Variables),
		"collection_count":     len(resp.Meta.VariableCollections),
	})
}
