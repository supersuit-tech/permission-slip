package linear

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// addLabelAction implements connectors.Action for linear.add_label.
// It adds a label to an issue by fetching current labels and appending.
type addLabelAction struct {
	conn *LinearConnector
}

type addLabelParams struct {
	IssueID string `json:"issue_id"`
	LabelID string `json:"label_id"`
}

func (p *addLabelParams) validate() error {
	if p.IssueID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: issue_id"}
	}
	if p.LabelID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: label_id"}
	}
	return nil
}

// getIssueLabelIDsQuery fetches current label IDs for an issue.
const getIssueLabelIDsQuery = `query GetIssueLabels($id: String!) {
	issue(id: $id) {
		labels {
			nodes {
				id
			}
		}
	}
}`

type getIssueLabelIDsResponse struct {
	Issue struct {
		Labels struct {
			Nodes []struct {
				ID string `json:"id"`
			} `json:"nodes"`
		} `json:"labels"`
	} `json:"issue"`
}

const addLabelMutation = `mutation IssueUpdate($id: String!, $input: IssueUpdateInput!) {
	issueUpdate(id: $id, input: $input) {
		success
		issue {
			id
			identifier
			labels {
				nodes {
					id
					name
				}
			}
		}
	}
}`

type addLabelResponse struct {
	IssueUpdate struct {
		Success bool `json:"success"`
		Issue   struct {
			ID         string `json:"id"`
			Identifier string `json:"identifier"`
			Labels     struct {
				Nodes []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"nodes"`
			} `json:"labels"`
		} `json:"issue"`
	} `json:"issueUpdate"`
}

func (a *addLabelAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params addLabelParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Fetch current label IDs to merge with the new one.
	var current getIssueLabelIDsResponse
	if err := a.conn.doGraphQL(ctx, req.Credentials, getIssueLabelIDsQuery, map[string]any{"id": params.IssueID}, &current); err != nil {
		return nil, err
	}

	labelIDs := make([]string, 0, len(current.Issue.Labels.Nodes)+1)
	alreadyHas := false
	for _, l := range current.Issue.Labels.Nodes {
		labelIDs = append(labelIDs, l.ID)
		if l.ID == params.LabelID {
			alreadyHas = true
		}
	}
	if !alreadyHas {
		labelIDs = append(labelIDs, params.LabelID)
	}

	vars := map[string]any{
		"id":    params.IssueID,
		"input": map[string]any{"labelIds": labelIDs},
	}

	var resp addLabelResponse
	if err := a.conn.doGraphQL(ctx, req.Credentials, addLabelMutation, vars, &resp); err != nil {
		return nil, err
	}

	if !resp.IssueUpdate.Success {
		return nil, &connectors.ExternalError{StatusCode: 200, Message: "Linear issueUpdate returned success=false"}
	}

	labels := make([]map[string]string, 0, len(resp.IssueUpdate.Issue.Labels.Nodes))
	for _, l := range resp.IssueUpdate.Issue.Labels.Nodes {
		labels = append(labels, map[string]string{"id": l.ID, "name": l.Name})
	}

	return connectors.JSONResult(map[string]interface{}{
		"id":         resp.IssueUpdate.Issue.ID,
		"identifier": resp.IssueUpdate.Issue.Identifier,
		"labels":     labels,
	})
}
