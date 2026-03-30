package firestore

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type setAction struct {
	conn *FirestoreConnector
}

type setParams struct {
	Path                 string                 `json:"path"`
	Data                 map[string]interface{} `json:"data"`
	Merge                *bool                  `json:"merge"`
	AllowedPaths         []string               `json:"allowed_paths"`
	AllowedWriteFields   []string               `json:"allowed_write_fields"`
}

func (p *setParams) validate() error {
	if err := validateDocumentPath(p.Path); err != nil {
		return err
	}
	if err := validateAllowedPaths(p.Path, p.AllowedPaths, "document"); err != nil {
		return err
	}
	if p.Data == nil {
		return &connectors.ValidationError{Message: "missing required parameter: data"}
	}
	if len(p.AllowedWriteFields) > 0 {
		if err := validateFieldAllowlist(p.AllowedWriteFields); err != nil {
			return err
		}
		return validateMapKeysSubset(p.Data, buildFieldSet(p.AllowedWriteFields), "data")
	}
	return nil
}

func (a *setAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params setParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	merge := params.Merge != nil && *params.Merge

	ctx, cancel := a.conn.withTimeout(ctx)
	defer cancel()

	runner, err := a.conn.openRunner(ctx, req.Credentials)
	if err != nil {
		return nil, err
	}
	defer func() { _ = runner.close() }()

	if err := runner.setDocument(ctx, params.Path, params.Data, merge); err != nil {
		return nil, err
	}
	return connectors.JSONResult(map[string]interface{}{
		"path":   params.Path,
		"status": "ok",
		"merge":  merge,
	})
}
