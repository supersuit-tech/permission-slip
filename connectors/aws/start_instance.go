package aws

import (
	"context"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// startInstanceAction implements connectors.Action for aws.start_instance.
type startInstanceAction struct {
	conn *AWSConnector
}

// Execute starts a stopped EC2 instance.
func (a *startInstanceAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	return executeInstanceStateChange(ctx, a.conn, req, "StartInstances")
}
