package tresorit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListFiles_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/my-tresor" {
			t.Errorf("path = %s, want /my-tresor", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Name>my-tresor</Name>
  <Prefix>docs/</Prefix>
  <IsTruncated>false</IsTruncated>
  <Contents>
    <Key>docs/report.pdf</Key>
    <LastModified>2024-01-01T00:00:00Z</LastModified>
    <Size>1024</Size>
  </Contents>
  <CommonPrefixes>
    <Prefix>docs/archive/</Prefix>
  </CommonPrefixes>
</ListBucketResult>`))
	}))
	defer srv.Close()

	conn := newTestConnector(&http.Client{
		Transport: &testTransport{inner: srv.Client().Transport, testURL: srv.URL},
	})
	action := &listFilesAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "tresorit.list_files",
		Parameters:  json.RawMessage(`{"tresor":"my-tresor","prefix":"docs/"}`),
		Credentials: validCredsWithEndpoint(srv.URL),
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
}

func TestListFiles_MissingTresor(t *testing.T) {
	t.Parallel()
	action := &listFilesAction{conn: New()}
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "tresorit.list_files",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}
