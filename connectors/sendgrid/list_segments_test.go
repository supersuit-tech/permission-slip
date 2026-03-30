package sendgrid

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListSegments_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/marketing/segments/2.0" {
			t.Errorf("path = %s, want /marketing/segments/2.0", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{
					"id":             "seg_1",
					"name":           "Active Users",
					"contacts_count": 500,
					"created_at":     "2026-01-01T00:00:00Z",
					"updated_at":     "2026-03-01T00:00:00Z",
				},
				{
					"id":             "seg_2",
					"name":           "Inactive Users",
					"contacts_count": 150,
					"created_at":     "2026-02-01T00:00:00Z",
					"updated_at":     "2026-03-05T00:00:00Z",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.list_segments"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "sendgrid.list_segments",
		Parameters:  json.RawMessage(`{}`),
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
	segments, ok := data["segments"].([]any)
	if !ok {
		t.Fatal("segments not present in result")
	}
	if len(segments) != 2 {
		t.Errorf("segments length = %d, want 2", len(segments))
	}
}

func TestListSegments_Empty(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.list_segments"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "sendgrid.list_segments",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["count"] != float64(0) {
		t.Errorf("count = %v, want 0", data["count"])
	}
}
