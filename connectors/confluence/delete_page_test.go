package confluence

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDeletePage_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/pages/123456" {
			t.Errorf("expected path /pages/123456, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &deletePageAction{conn: conn}

	params, _ := json.Marshal(deletePageParams{PageID: "123456"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.delete_page",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["id"] != "123456" {
		t.Errorf("expected id '123456', got %q", data["id"])
	}
	if data["status"] != "deleted" {
		t.Errorf("expected status 'deleted', got %q", data["status"])
	}
}

func TestDeletePage_MissingPageID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deletePageAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.delete_page",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}
