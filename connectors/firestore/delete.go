package firestore

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type deleteAction struct {
	conn *FirestoreConnector
}

type deleteParams struct {
	Path         string   `json:"path"`
	AllowedPaths []string `json:"allowed_paths"`
}

func (p *deleteParams) validate() error {
	if err := validateDocumentPath(p.Path); err != nil {
		return err
	}
	return validateAllowedPaths(p.Path, p.AllowedPaths, "document")
}

func (a *deleteAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	ctx, cancel := a.conn.withTimeout(ctx)
	defer cancel()

	runner, err := a.conn.openRunner(ctx, req.Credentials)
	if err != nil {
		return nil, err
	}
	defer func() { _ = runner.close() }()

	if err := runner.deleteDocument(ctx, params.Path); err != nil {
		return nil, err
	}
	return connectors.JSONResult(map[string]interface{}{
		"path":   params.Path,
		"status": "deleted",
	})
}
