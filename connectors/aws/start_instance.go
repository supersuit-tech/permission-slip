package aws

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// startInstanceAction implements connectors.Action for aws.start_instance.
type startInstanceAction struct {
	conn *AWSConnector
}

type startInstancesResponse struct {
	XMLName      xml.Name `xml:"StartInstancesResponse"`
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

// Execute starts a stopped EC2 instance.
func (a *startInstanceAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[instanceIDParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	body := buildEC2InstanceActionBody("StartInstances", params.InstanceID)
	respBody, err := a.conn.do(ctx, req.Credentials, "POST", ec2Host(params.Region), "/", "", body)
	if err != nil {
		return nil, err
	}

	var xmlResp startInstancesResponse
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
