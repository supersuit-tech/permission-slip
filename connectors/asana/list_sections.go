package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type listSectionsAction struct {
	conn *AsanaConnector
}

type listSectionsParams struct {
	ProjectID string `json:"project_id"`
}

func (p *listSectionsParams) validate() error {
	if p.ProjectID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: project_id"}
	}
	return nil
}

func (a *listSectionsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listSectionsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	fullURL := fmt.Sprintf("%s/projects/%s/sections", a.conn.baseURL, url.PathEscape(params.ProjectID))

	var envelope struct {
		Data []struct {
			GID  string `json:"gid"`
			Name string `json:"name"`
		} `json:"data"`
	}

	if err := a.conn.doRaw(ctx, req.Credentials, "GET", fullURL, &envelope); err != nil {
		return nil, err
	}

	return connectors.JSONResult(envelope.Data)
}
