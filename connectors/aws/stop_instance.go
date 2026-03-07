package aws

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// stopInstanceAction implements connectors.Action for aws.stop_instance.
type stopInstanceAction struct {
	conn *AWSConnector
}

type stopInstancesResponse struct {
	XMLName      xml.Name `xml:"StopInstancesResponse"`
	InstancesSet []struct {
		InstanceID   string `xml:"instanceId"`
		CurrentState struct {
			Code int    `xml:"code"`
			Name string `xml:"name"`
		} `xml:"currentState"`
		PreviousState struct {
			Code int    `xml:"code"`
			Name string `xml:"name"`
		} `xml:"previousState"`
	} `xml:"instancesSet>item"`
}

// Execute stops a running EC2 instance.
func (a *stopInstanceAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[instanceIDParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	body := buildEC2InstanceActionBody("StopInstances", params.InstanceID)
	respBody, err := a.conn.do(ctx, req.Credentials, "POST", ec2Host(params.Region), "/", "", body)
	if err != nil {
		return nil, err
	}

	var xmlResp stopInstancesResponse
	if err := xml.Unmarshal(respBody, &xmlResp); err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("parsing EC2 response: %v", err)}
	}

	if len(xmlResp.InstancesSet) == 0 {
		return nil, &connectors.ExternalError{Message: "no instance state change returned"}
	}

	inst := xmlResp.InstancesSet[0]
	return connectors.JSONResult(map[string]any{
		"instance_id":    inst.InstanceID,
		"current_state":  inst.CurrentState.Name,
		"previous_state": inst.PreviousState.Name,
	})
}
