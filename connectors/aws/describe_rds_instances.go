package aws

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// describeRDSInstancesAction implements connectors.Action for aws.describe_rds_instances.
type describeRDSInstancesAction struct {
	conn *AWSConnector
}

type describeRDSParams struct {
	Region       string `json:"region"`
	DBInstanceID string `json:"db_instance_id"`
}

func (p *describeRDSParams) validate() error {
	if p.Region == "" {
		return &connectors.ValidationError{Message: "missing required parameter: region"}
	}
	return nil
}

type describeDBInstancesResponse struct {
	XMLName xml.Name `xml:"DescribeDBInstancesResponse"`
	Result  struct {
		DBInstances []struct {
			DBInstanceID         string `xml:"DBInstanceIdentifier"`
			DBInstanceClass      string `xml:"DBInstanceClass"`
			Engine               string `xml:"Engine"`
			EngineVersion        string `xml:"EngineVersion"`
			DBInstanceStatus     string `xml:"DBInstanceStatus"`
			MasterUsername       string `xml:"MasterUsername"`
			AllocatedStorage     int    `xml:"AllocatedStorage"`
			AvailabilityZone     string `xml:"AvailabilityZone"`
			MultiAZ              bool   `xml:"MultiAZ"`
			StorageType          string `xml:"StorageType"`
			DBInstanceArn        string `xml:"DBInstanceArn"`
			InstanceCreateTime   string `xml:"InstanceCreateTime"`
			PubliclyAccessible   bool   `xml:"PubliclyAccessible"`
			StorageEncrypted     bool   `xml:"StorageEncrypted"`
			Endpoint             struct {
				Address string `xml:"Address"`
				Port    int    `xml:"Port"`
			} `xml:"Endpoint"`
		} `xml:"DBInstances>DBInstance"`
	} `xml:"DescribeDBInstancesResult"`
}

type rdsInstanceInfo struct {
	DBInstanceID       string `json:"db_instance_id"`
	DBInstanceClass    string `json:"db_instance_class"`
	Engine             string `json:"engine"`
	EngineVersion      string `json:"engine_version"`
	Status             string `json:"status"`
	AllocatedStorage   int    `json:"allocated_storage_gb"`
	AvailabilityZone   string `json:"availability_zone,omitempty"`
	MultiAZ            bool   `json:"multi_az"`
	StorageType        string `json:"storage_type"`
	EndpointAddress    string `json:"endpoint_address,omitempty"`
	EndpointPort       int    `json:"endpoint_port,omitempty"`
	PubliclyAccessible bool   `json:"publicly_accessible"`
	StorageEncrypted   bool   `json:"storage_encrypted"`
	CreatedAt          string `json:"created_at,omitempty"`
}

// Execute describes RDS instances via the RDS Query API.
func (a *describeRDSInstancesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params describeRDSParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	queryParams := url.Values{}
	queryParams.Set("Action", "DescribeDBInstances")
	queryParams.Set("Version", "2014-10-31")
	if params.DBInstanceID != "" {
		queryParams.Set("DBInstanceIdentifier", params.DBInstanceID)
	}

	host := fmt.Sprintf("rds.%s.amazonaws.com", params.Region)
	body := []byte(queryParams.Encode())

	respBody, err := a.conn.do(ctx, req.Credentials, "POST", host, "/", "", body)
	if err != nil {
		return nil, err
	}

	var xmlResp describeDBInstancesResponse
	if err := xml.Unmarshal(respBody, &xmlResp); err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("parsing RDS response: %v", err)}
	}

	instances := make([]rdsInstanceInfo, 0, len(xmlResp.Result.DBInstances))
	for _, db := range xmlResp.Result.DBInstances {
		instances = append(instances, rdsInstanceInfo{
			DBInstanceID:       db.DBInstanceID,
			DBInstanceClass:    db.DBInstanceClass,
			Engine:             db.Engine,
			EngineVersion:      db.EngineVersion,
			Status:             db.DBInstanceStatus,
			AllocatedStorage:   db.AllocatedStorage,
			AvailabilityZone:   db.AvailabilityZone,
			MultiAZ:            db.MultiAZ,
			StorageType:        db.StorageType,
			EndpointAddress:    db.Endpoint.Address,
			EndpointPort:       db.Endpoint.Port,
			PubliclyAccessible: db.PubliclyAccessible,
			StorageEncrypted:   db.StorageEncrypted,
			CreatedAt:          db.InstanceCreateTime,
		})
	}

	return connectors.JSONResult(map[string]any{
		"instances": instances,
		"count":     len(instances),
		"region":    params.Region,
	})
}
