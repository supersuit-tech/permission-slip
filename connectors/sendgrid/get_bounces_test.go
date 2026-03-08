package sendgrid

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetBounces_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/suppression/bounces" {
			t.Errorf("got %s %s, want GET /suppression/bounces", r.Method, r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{
			{"email": "bounced@example.com", "reason": "550 No such user", "status": "5.1.1", "created": 1700000000},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.get_bounces"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "sendgrid.get_bounces",
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
	if data["count"] != float64(1) {
		t.Errorf("count = %v, want 1", data["count"])
	}
}

func TestGetBounces_WithTimeRange(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("start_time") == "" {
			t.Error("expected start_time query param")
		}
		if q.Get("end_time") == "" {
			t.Error("expected end_time query param")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.get_bounces"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "sendgrid.get_bounces",
		Parameters:  json.RawMessage(`{"start_time":"2026-01-01T00:00:00Z","end_time":"2026-01-31T23:59:59Z"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestGetBounces_ValidationErrors(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["sendgrid.get_bounces"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "invalid start_time", params: `{"start_time":"not-a-date"}`},
		{name: "invalid end_time", params: `{"end_time":"not-a-date"}`},
		{name: "negative limit", params: `{"limit":-1}`},
		{name: "negative offset", params: `{"offset":-1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "sendgrid.get_bounces",
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
