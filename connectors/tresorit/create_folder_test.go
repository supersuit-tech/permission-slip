package tresorit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateFolder_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/my-tresor/projects/new/" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	conn := newTestConnector(&http.Client{
		Transport: &testTransport{inner: srv.Client().Transport, testURL: srv.URL},
	})
	action := &createFolderAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "tresorit.create_folder",
		Parameters:  json.RawMessage(`{"tresor":"my-tresor","path":"projects/new"}`),
		Credentials: validCredsWithEndpoint(srv.URL),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestDeleteFile_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/my-tresor/docs/old.txt" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newTestConnector(&http.Client{
		Transport: &testTransport{inner: srv.Client().Transport, testURL: srv.URL},
	})
	action := &deleteFileAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "tresorit.delete_file",
		Parameters:  json.RawMessage(`{"tresor":"my-tresor","key":"docs/old.txt"}`),
		Credentials: validCredsWithEndpoint(srv.URL),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if data["deleted"] != true {
		t.Errorf("deleted = %v", data["deleted"])
	}
}
