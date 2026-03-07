package aws

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// describeInstancesAction implements connectors.Action for aws.describe_instances.
type describeInstancesAction struct {
	conn *AWSConnector
}

type describeInstancesParams struct {
	Region      string   `json:"region"`
	InstanceIDs []string `json:"instance_ids"`
}

func (p *describeInstancesParams) validate() error {
	return validateRegion(p.Region)
}

// describeInstancesResponse represents the EC2 DescribeInstances XML response.
type describeInstancesResponse struct {
	XMLName        xml.Name `xml:"DescribeInstancesResponse"`
	ReservationSet []struct {
		InstancesSet []struct {
			InstanceID   string `xml:"instanceId"`
			InstanceType string `xml:"instanceType"`
			State        struct {
				Code int    `xml:"code"`
				Name string `xml:"name"`
			} `xml:"instanceState"`
			PrivateIP  string `xml:"privateIpAddress"`
			PublicIP   string `xml:"ipAddress"`
			LaunchTime string `xml:"launchTime"`
			Placement  struct {
				AvailabilityZone string `xml:"availabilityZone"`
			} `xml:"placement"`
			TagSet []struct {
				Key   string `xml:"key"`
				Value string `xml:"value"`
			} `xml:"tagSet>item"`
		} `xml:"instancesSet>item"`
	} `xml:"reservationSet>item"`
}

type instanceInfo struct {
	InstanceID       string            `json:"instance_id"`
	InstanceType     string            `json:"instance_type"`
	State            string            `json:"state"`
	PrivateIP        string            `json:"private_ip,omitempty"`
	PublicIP         string            `json:"public_ip,omitempty"`
	AvailabilityZone string            `json:"availability_zone,omitempty"`
	LaunchTime       string            `json:"launch_time,omitempty"`
	Tags             map[string]string `json:"tags,omitempty"`
}

// Execute describes EC2 instances via the EC2 Query API.
func (a *describeInstancesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[describeInstancesParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	queryParams := url.Values{}
	queryParams.Set("Action", "DescribeInstances")
	queryParams.Set("Version", "2016-11-15")
	for i, id := range params.InstanceIDs {
		queryParams.Set(fmt.Sprintf("InstanceId.%d", i+1), id)
	}

	host := fmt.Sprintf("ec2.%s.amazonaws.com", params.Region)
	body := []byte(queryParams.Encode())

	respBody, err := a.conn.do(ctx, req.Credentials, "POST", host, "/", "", body)
	if err != nil {
		return nil, err
	}

	var xmlResp describeInstancesResponse
	if err := xml.Unmarshal(respBody, &xmlResp); err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("parsing EC2 response: %v", err)}
	}

	var instances []instanceInfo
	for _, reservation := range xmlResp.ReservationSet {
		for _, inst := range reservation.InstancesSet {
			info := instanceInfo{
				InstanceID:       inst.InstanceID,
				InstanceType:     inst.InstanceType,
				State:            inst.State.Name,
				PrivateIP:        inst.PrivateIP,
				PublicIP:         inst.PublicIP,
				AvailabilityZone: inst.Placement.AvailabilityZone,
				LaunchTime:       inst.LaunchTime,
			}
			if len(inst.TagSet) > 0 {
				info.Tags = make(map[string]string, len(inst.TagSet))
				for _, tag := range inst.TagSet {
					info.Tags[tag.Key] = tag.Value
				}
			}
			instances = append(instances, info)
		}
	}

	return connectors.JSONResult(map[string]any{
		"instances": instances,
		"count":     len(instances),
		"region":    params.Region,
	})
}

// buildEC2InstanceActionBody builds a form-encoded body for single-instance EC2 actions.
func buildEC2InstanceActionBody(action, instanceID string) []byte {
	params := url.Values{}
	params.Set("Action", action)
	params.Set("Version", "2016-11-15")
	params.Set("InstanceId.1", instanceID)
	return []byte(params.Encode())
}

// ec2Host returns the EC2 API hostname for the given region.
func ec2Host(region string) string {
	return fmt.Sprintf("ec2.%s.amazonaws.com", region)
}

// instanceIDParams is shared by start/stop/restart instance actions.
type instanceIDParams struct {
	Region     string `json:"region"`
	InstanceID string `json:"instance_id"`
}

func (p *instanceIDParams) validate() error {
	if err := validateRegion(p.Region); err != nil {
		return err
	}
	return validateInstanceID(p.InstanceID)
}
