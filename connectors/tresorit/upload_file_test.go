package tresorit

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestUploadFile_Success(t *testing.T) {
	t.Parallel()

	content := base64.StdEncoding.EncodeToString([]byte("upload payload"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "upload payload" {
			t.Errorf("body = %q", string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	conn := newTestConnector(&http.Client{
		Transport: &testTransport{inner: srv.Client().Transport, testURL: srv.URL},
	})
	action := &uploadFileAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "tresorit.upload_file",
		Parameters:  json.RawMessage(`{"tresor":"my-tresor","key":"docs/new.txt","content":"` + content + `"}`),
		Credentials: validCredsWithEndpoint(srv.URL),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if data["size"] != float64(len("upload payload")) {
		t.Errorf("size = %v", data["size"])
	}
}

func TestUploadFile_InvalidBase64(t *testing.T) {
	t.Parallel()
	action := &uploadFileAction{conn: New()}
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "tresorit.upload_file",
		Parameters:  json.RawMessage(`{"tresor":"my-tresor","key":"x","content":"not-base64!!!"}`),
		Credentials: validCreds(),
	})
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}
