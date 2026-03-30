package aws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListS3Objects_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Name>my-bucket</Name>
  <Prefix>logs/</Prefix>
  <IsTruncated>false</IsTruncated>
  <Contents>
    <Key>logs/app.log</Key>
    <LastModified>2024-01-01T00:00:00Z</LastModified>
    <Size>1024</Size>
    <StorageClass>STANDARD</StorageClass>
    <ETag>"abc123"</ETag>
  </Contents>
  <Contents>
    <Key>logs/error.log</Key>
    <LastModified>2024-01-02T00:00:00Z</LastModified>
    <Size>2048</Size>
    <StorageClass>STANDARD</StorageClass>
    <ETag>"def456"</ETag>
  </Contents>
</ListBucketResult>`))
	}))
	defer srv.Close()

	conn := &AWSConnector{client: &http.Client{
		Transport: &testTransport{inner: srv.Client().Transport, testURL: srv.URL},
	}}
	action := &listS3ObjectsAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "aws.list_s3_objects",
		Parameters:  json.RawMessage(`{"region":"us-east-1","bucket":"my-bucket","prefix":"logs/"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["count"] != float64(2) {
		t.Errorf("count = %v, want 2", data["count"])
	}
	if data["bucket"] != "my-bucket" {
		t.Errorf("bucket = %v, want my-bucket", data["bucket"])
	}
}

func TestListS3Objects_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["aws.list_s3_objects"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing region", params: `{"bucket":"my-bucket"}`},
		{name: "missing bucket", params: `{"region":"us-east-1"}`},
		{name: "invalid max_keys", params: `{"region":"us-east-1","bucket":"my-bucket","max_keys":1001}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "aws.list_s3_objects",
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
