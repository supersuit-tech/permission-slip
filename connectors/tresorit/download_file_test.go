package tresorit

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestDownloadFile_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/my-tresor/docs/report.pdf" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello tresorit"))
	}))
	defer srv.Close()

	conn := newTestConnector(&http.Client{
		Transport: &testTransport{inner: srv.Client().Transport, testURL: srv.URL},
	})
	action := &downloadFileAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "tresorit.download_file",
		Parameters:  json.RawMessage(`{"tresor":"my-tresor","key":"docs/report.pdf"}`),
		Credentials: validCredsWithEndpoint(srv.URL),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got, _ := base64.StdEncoding.DecodeString(data["content"].(string))
	if string(got) != "hello tresorit" {
		t.Errorf("content = %q", string(got))
	}
}

func TestDownloadFile_MissingKey(t *testing.T) {
	t.Parallel()
	action := &downloadFileAction{conn: New()}
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "tresorit.download_file",
		Parameters:  json.RawMessage(`{"tresor":"my-tresor"}`),
		Credentials: validCreds(),
	})
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}
