package aws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetMetrics_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<GetMetricStatisticsResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
  <GetMetricStatisticsResult>
    <Label>CPUUtilization</Label>
    <Datapoints>
      <member>
        <Timestamp>2024-01-01T00:00:00Z</Timestamp>
        <Average>25.5</Average>
        <Unit>Percent</Unit>
      </member>
      <member>
        <Timestamp>2024-01-01T00:05:00Z</Timestamp>
        <Average>30.2</Average>
        <Unit>Percent</Unit>
      </member>
    </Datapoints>
  </GetMetricStatisticsResult>
</GetMetricStatisticsResponse>`))
	}))
	defer srv.Close()

	conn := &AWSConnector{client: &http.Client{
		Transport: &testTransport{inner: srv.Client().Transport, testURL: srv.URL},
	}}
	action := &getMetricsAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "aws.get_metrics",
		Parameters: json.RawMessage(`{
			"region": "us-east-1",
			"namespace": "AWS/EC2",
			"metric_name": "CPUUtilization",
			"start_time": "2024-01-01T00:00:00Z",
			"end_time": "2024-01-01T01:00:00Z",
			"period": 300
		}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["label"] != "CPUUtilization" {
		t.Errorf("label = %v, want CPUUtilization", data["label"])
	}
	datapoints, ok := data["datapoints"].([]any)
	if !ok || len(datapoints) != 2 {
		t.Fatalf("expected 2 datapoints, got %v", data["datapoints"])
	}
}

func TestGetMetrics_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["aws.get_metrics"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing region", params: `{"namespace":"AWS/EC2","metric_name":"CPU","start_time":"2024-01-01T00:00:00Z","end_time":"2024-01-01T01:00:00Z","period":300}`},
		{name: "missing namespace", params: `{"region":"us-east-1","metric_name":"CPU","start_time":"2024-01-01T00:00:00Z","end_time":"2024-01-01T01:00:00Z","period":300}`},
		{name: "missing metric_name", params: `{"region":"us-east-1","namespace":"AWS/EC2","start_time":"2024-01-01T00:00:00Z","end_time":"2024-01-01T01:00:00Z","period":300}`},
		{name: "missing start_time", params: `{"region":"us-east-1","namespace":"AWS/EC2","metric_name":"CPU","end_time":"2024-01-01T01:00:00Z","period":300}`},
		{name: "missing end_time", params: `{"region":"us-east-1","namespace":"AWS/EC2","metric_name":"CPU","start_time":"2024-01-01T00:00:00Z","period":300}`},
		{name: "missing period", params: `{"region":"us-east-1","namespace":"AWS/EC2","metric_name":"CPU","start_time":"2024-01-01T00:00:00Z","end_time":"2024-01-01T01:00:00Z"}`},
		{name: "invalid stat", params: `{"region":"us-east-1","namespace":"AWS/EC2","metric_name":"CPU","start_time":"2024-01-01T00:00:00Z","end_time":"2024-01-01T01:00:00Z","period":300,"stat":"Invalid"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "aws.get_metrics",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("Execute() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}
