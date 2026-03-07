package make

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestToggleScenario_Enable(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/scenarios/42" {
			t.Errorf("expected path /scenarios/42, got %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		scheduling, ok := body["scheduling"].(map[string]any)
		if !ok {
			t.Fatal("expected scheduling in request body")
		}
		if scheduling["isEnabled"] != true {
			t.Errorf("expected isEnabled true, got %v", scheduling["isEnabled"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"scenario": map[string]any{"id": 42, "isEnabled": true},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &toggleScenarioAction{conn: conn}

	params, _ := json.Marshal(toggleScenarioParams{ScenarioID: 42, Enabled: true})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "make.toggle_scenario",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestToggleScenario_Disable(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		scheduling := body["scheduling"].(map[string]any)
		if scheduling["isEnabled"] != false {
			t.Errorf("expected isEnabled false, got %v", scheduling["isEnabled"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"scenario": map[string]any{"id": 42, "isEnabled": false},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &toggleScenarioAction{conn: conn}

	params, _ := json.Marshal(toggleScenarioParams{ScenarioID: 42, Enabled: false})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "make.toggle_scenario",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestToggleScenario_InvalidScenarioID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &toggleScenarioAction{conn: conn}

	params, _ := json.Marshal(toggleScenarioParams{ScenarioID: 0, Enabled: true})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "make.toggle_scenario",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid scenario_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestToggleScenario_Forbidden(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"Forbidden"}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &toggleScenarioAction{conn: conn}

	params, _ := json.Marshal(toggleScenarioParams{ScenarioID: 42, Enabled: true})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "make.toggle_scenario",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for forbidden")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}
