package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listPipelinesAction implements connectors.Action for hubspot.list_pipelines.
// It fetches deal pipelines via GET /crm/v3/pipelines/deals.
type listPipelinesAction struct {
	conn *HubSpotConnector
}

type listPipelinesParams struct {
	ObjectType string `json:"object_type"`
}

type pipelineStage struct {
	ID          string            `json:"id"`
	Label       string            `json:"label"`
	DisplayOrder int              `json:"displayOrder"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type pipeline struct {
	ID           string          `json:"id"`
	Label        string          `json:"label"`
	DisplayOrder int             `json:"displayOrder"`
	Stages       []pipelineStage `json:"stages"`
}

type pipelinesResponse struct {
	Results []pipeline `json:"results"`
}

func (a *listPipelinesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listPipelinesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	objectType := params.ObjectType
	if objectType == "" {
		objectType = "deals"
	}
	validPipelineTypes := map[string]bool{"deals": true, "tickets": true}
	if !validPipelineTypes[objectType] {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid object_type %q: must be deals or tickets", objectType)}
	}

	path := fmt.Sprintf("/crm/v3/pipelines/%s", objectType)
	var resp pipelinesResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
