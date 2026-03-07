package calendly

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListEventTypes_Success(t *testing.T) {
	t.Parallel()

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First call is /users/me
			if r.URL.Path != "/users/me" {
				t.Errorf("expected path /users/me, got %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(usersmeResponse{
				Resource: struct {
					URI  string `json:"uri"`
					Name string `json:"name"`
				}{URI: "https://api.calendly.com/users/abc123", Name: "Test User"},
			})
			return
		}
		// Second call is /event_types
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/event_types" {
			t.Errorf("expected path /event_types, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("user") != "https://api.calendly.com/users/abc123" {
			t.Errorf("expected user query param, got %s", r.URL.Query().Get("user"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(calendlyEventTypesResponse{
			Collection: []calendlyEventType{
				{
					URI:              "https://api.calendly.com/event_types/et1",
					Name:             "30 Minute Meeting",
					Active:           true,
					Slug:             "30min",
					SchedulingURL:    "https://calendly.com/testuser/30min",
					Duration:         30,
					Kind:             "solo",
					DescriptionPlain: "A quick 30 minute chat",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listEventTypesAction{conn: conn}

	params, _ := json.Marshal(listEventTypesParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "calendly.list_event_types",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		TotalEventTypes int             `json:"total_event_types"`
		EventTypes      []eventTypeItem `json:"event_types"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.EventTypes) != 1 {
		t.Fatalf("expected 1 event type, got %d", len(data.EventTypes))
	}
	if data.EventTypes[0].Name != "30 Minute Meeting" {
		t.Errorf("expected name '30 Minute Meeting', got %q", data.EventTypes[0].Name)
	}
	if data.EventTypes[0].Duration != 30 {
		t.Errorf("expected duration 30, got %d", data.EventTypes[0].Duration)
	}
}

func TestListEventTypes_EmptyResult(t *testing.T) {
	t.Parallel()

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(usersmeResponse{
				Resource: struct {
					URI  string `json:"uri"`
					Name string `json:"name"`
				}{URI: "https://api.calendly.com/users/abc123", Name: "Test User"},
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(calendlyEventTypesResponse{Collection: nil})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listEventTypesAction{conn: conn}

	params, _ := json.Marshal(listEventTypesParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "calendly.list_event_types",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		TotalEventTypes int             `json:"total_event_types"`
		EventTypes      []eventTypeItem `json:"event_types"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.EventTypes) != 0 {
		t.Errorf("expected 0 event types, got %d", len(data.EventTypes))
	}
}

func TestListEventTypes_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(calendlyAPIError{Title: "Unauthenticated", Message: "Invalid token."})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listEventTypesAction{conn: conn}

	params, _ := json.Marshal(listEventTypesParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "calendly.list_event_types",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}

func TestListEventTypes_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listEventTypesAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "calendly.list_event_types",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
