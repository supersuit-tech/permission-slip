package aws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestStartInstance_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<StartInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
  <instancesSet>
    <item>
      <instanceId>i-1234567890abcdef0</instanceId>
      <currentState>
        <code>0</code>
        <name>pending</name>
      </currentState>
      <previousState>
        <code>80</code>
        <name>stopped</name>
      </previousState>
    </item>
  </instancesSet>
</StartInstancesResponse>`))
	}))
	defer srv.Close()

	conn := &AWSConnector{client: &http.Client{
		Transport: &testTransport{inner: srv.Client().Transport, testURL: srv.URL},
	}}
	action := &startInstanceAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "aws.start_instance",
		Parameters:  json.RawMessage(`{"region":"us-east-1","instance_id":"i-1234567890abcdef0"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["instance_id"] != "i-1234567890abcdef0" {
		t.Errorf("instance_id = %v, want i-1234567890abcdef0", data["instance_id"])
	}
	if data["current_state"] != "pending" {
		t.Errorf("current_state = %v, want pending", data["current_state"])
	}
	if data["previous_state"] != "stopped" {
		t.Errorf("previous_state = %v, want stopped", data["previous_state"])
	}
}

func TestStartInstance_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["aws.start_instance"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing region", params: `{"instance_id":"i-123"}`},
		{name: "missing instance_id", params: `{"region":"us-east-1"}`},
		{name: "invalid instance_id", params: `{"region":"us-east-1","instance_id":"bad"}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "aws.start_instance",
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

func TestStopInstance_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["aws.stop_instance"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing region", params: `{"instance_id":"i-123"}`},
		{name: "missing instance_id", params: `{"region":"us-east-1"}`},
		{name: "invalid instance_id", params: `{"region":"us-east-1","instance_id":"bad"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "aws.stop_instance",
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

func TestRestartInstance_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["aws.restart_instance"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing region", params: `{"instance_id":"i-123"}`},
		{name: "missing instance_id", params: `{"region":"us-east-1"}`},
		{name: "invalid instance_id", params: `{"region":"us-east-1","instance_id":"bad"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "aws.restart_instance",
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
