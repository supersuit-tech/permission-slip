package aws

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// restartInstanceAction implements connectors.Action for aws.restart_instance.
type restartInstanceAction struct {
	conn *AWSConnector
}

// Execute reboots a running EC2 instance. The RebootInstances API returns
// no body on success (just HTTP 200), so we return a simple confirmation.
func (a *restartInstanceAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params instanceIDParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := buildEC2InstanceActionBody("RebootInstances", params.InstanceID)
	_, err := a.conn.do(ctx, req.Credentials, "POST", ec2Host(params.Region), "/", "", body)
	if err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"instance_id": params.InstanceID,
		"status":      "rebooting",
	})
}
