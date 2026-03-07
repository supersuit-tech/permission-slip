package aws

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreatePresignedURL_GET(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["aws.create_presigned_url"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "aws.create_presigned_url",
		Parameters:  json.RawMessage(`{"region":"us-east-1","bucket":"my-bucket","key":"path/to/file.txt","operation":"GET"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}

	url, ok := data["url"].(string)
	if !ok || url == "" {
		t.Fatal("url is missing or empty")
	}
	if !strings.Contains(url, "s3.us-east-1.amazonaws.com") {
		t.Errorf("url should contain s3.us-east-1.amazonaws.com, got %s", url)
	}
	if !strings.Contains(url, "/my-bucket/path/to/file.txt") {
		t.Errorf("url should contain the object path, got %s", url)
	}
	if !strings.Contains(url, "X-Amz-Signature=") {
		t.Errorf("url should contain X-Amz-Signature, got %s", url)
	}
	if data["operation"] != "GET" {
		t.Errorf("operation = %v, want GET", data["operation"])
	}
	if data["expires_in"] != float64(3600) {
		t.Errorf("expires_in = %v, want 3600", data["expires_in"])
	}
}

func TestCreatePresignedURL_PUT(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["aws.create_presigned_url"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "aws.create_presigned_url",
		Parameters:  json.RawMessage(`{"region":"us-west-2","bucket":"uploads","key":"data.csv","operation":"PUT","expires_in":7200}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["operation"] != "PUT" {
		t.Errorf("operation = %v, want PUT", data["operation"])
	}
	if data["expires_in"] != float64(7200) {
		t.Errorf("expires_in = %v, want 7200", data["expires_in"])
	}
}

func TestCreatePresignedURL_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["aws.create_presigned_url"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing region", params: `{"bucket":"b","key":"k","operation":"GET"}`},
		{name: "missing bucket", params: `{"region":"us-east-1","key":"k","operation":"GET"}`},
		{name: "missing key", params: `{"region":"us-east-1","bucket":"b","operation":"GET"}`},
		{name: "missing operation", params: `{"region":"us-east-1","bucket":"b","key":"k"}`},
		{name: "invalid operation", params: `{"region":"us-east-1","bucket":"b","key":"k","operation":"DELETE"}`},
		{name: "expires_in too large", params: `{"region":"us-east-1","bucket":"b","key":"k","operation":"GET","expires_in":999999}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "aws.create_presigned_url",
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
