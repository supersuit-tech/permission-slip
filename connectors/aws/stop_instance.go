package aws

import (
	"context"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// stopInstanceAction implements connectors.Action for aws.stop_instance.
type stopInstanceAction struct {
	conn *AWSConnector
}

// Execute stops a running EC2 instance.
func (a *stopInstanceAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	return executeInstanceStateChange(ctx, a.conn, req, "StopInstances")
}
