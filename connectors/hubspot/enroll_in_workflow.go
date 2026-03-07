package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// enrollInWorkflowAction implements connectors.Action for hubspot.enroll_in_workflow.
// It enrolls a contact in an automation workflow via POST /automation/v4/flows/{flowId}/enrollments.
type enrollInWorkflowAction struct {
	conn *HubSpotConnector
}

type enrollInWorkflowParams struct {
	FlowID    string `json:"flow_id"`
	ContactID string `json:"contact_id"`
}

func (p *enrollInWorkflowParams) validate() error {
	if p.FlowID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: flow_id"}
	}
	if !isValidHubSpotID(p.FlowID) {
		return &connectors.ValidationError{Message: "flow_id must be a numeric HubSpot ID"}
	}
	if p.ContactID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: contact_id"}
	}
	if !isValidHubSpotID(p.ContactID) {
		return &connectors.ValidationError{Message: "contact_id must be a numeric HubSpot ID"}
	}
	return nil
}

// enrollmentRequest is the request body for the workflow enrollment API.
type enrollmentRequest struct {
	ObjectID   string `json:"objectId"`
	ObjectType string `json:"objectType"`
}

// enrollmentResponse captures the response from the enrollment API.
type enrollmentResponse struct {
	ID         string `json:"id,omitempty"`
	Status     string `json:"status,omitempty"`
	ObjectID   string `json:"objectId,omitempty"`
	ObjectType string `json:"objectType,omitempty"`
}

func (a *enrollInWorkflowAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params enrollInWorkflowParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := enrollmentRequest{
		ObjectID:   params.ContactID,
		ObjectType: "CONTACT",
	}

	var resp enrollmentResponse
	path := fmt.Sprintf("/automation/v4/flows/%s/enrollments", params.FlowID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
