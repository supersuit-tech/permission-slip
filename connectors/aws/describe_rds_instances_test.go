package aws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDescribeRDSInstances_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<DescribeDBInstancesResponse xmlns="http://rds.amazonaws.com/doc/2014-10-31/">
  <DescribeDBInstancesResult>
    <DBInstances>
      <DBInstance>
        <DBInstanceIdentifier>my-db</DBInstanceIdentifier>
        <DBInstanceClass>db.t3.micro</DBInstanceClass>
        <Engine>postgres</Engine>
        <EngineVersion>15.4</EngineVersion>
        <DBInstanceStatus>available</DBInstanceStatus>
        <AllocatedStorage>20</AllocatedStorage>
        <AvailabilityZone>us-east-1a</AvailabilityZone>
        <MultiAZ>false</MultiAZ>
        <StorageType>gp3</StorageType>
        <PubliclyAccessible>false</PubliclyAccessible>
        <StorageEncrypted>true</StorageEncrypted>
        <Endpoint>
          <Address>my-db.abc123.us-east-1.rds.amazonaws.com</Address>
          <Port>5432</Port>
        </Endpoint>
      </DBInstance>
    </DBInstances>
  </DescribeDBInstancesResult>
</DescribeDBInstancesResponse>`))
	}))
	defer srv.Close()

	conn := &AWSConnector{client: &http.Client{
		Transport: &testTransport{inner: srv.Client().Transport, testURL: srv.URL},
	}}
	action := &describeRDSInstancesAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "aws.describe_rds_instances",
		Parameters:  json.RawMessage(`{"region":"us-east-1"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["count"] != float64(1) {
		t.Errorf("count = %v, want 1", data["count"])
	}
	instances, ok := data["instances"].([]any)
	if !ok || len(instances) != 1 {
		t.Fatalf("expected 1 instance, got %v", data["instances"])
	}
	inst := instances[0].(map[string]any)
	if inst["db_instance_id"] != "my-db" {
		t.Errorf("db_instance_id = %v, want my-db", inst["db_instance_id"])
	}
	if inst["engine"] != "postgres" {
		t.Errorf("engine = %v, want postgres", inst["engine"])
	}
	if inst["status"] != "available" {
		t.Errorf("status = %v, want available", inst["status"])
	}
}

func TestDescribeRDSInstances_MissingRegion(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["aws.describe_rds_instances"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "aws.describe_rds_instances",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
