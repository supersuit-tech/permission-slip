package make

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetScenario_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/scenarios/42" {
			t.Errorf("expected path /scenarios/42, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"scenario": map[string]any{
				"id":        42,
				"name":      "My Scenario",
				"isEnabled": true,
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getScenarioAction{conn: conn}

	params, _ := json.Marshal(getScenarioParams{ScenarioID: 42})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "make.get_scenario",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	scenario, ok := data["scenario"].(map[string]any)
	if !ok {
		t.Fatal("expected scenario in response")
	}
	if scenario["name"] != "My Scenario" {
		t.Errorf("expected name 'My Scenario', got %v", scenario["name"])
	}
}

func TestGetScenario_InvalidScenarioID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getScenarioAction{conn: conn}

	params, _ := json.Marshal(getScenarioParams{ScenarioID: -1})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "make.get_scenario",
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

func TestGetScenario_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Scenario not found"}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getScenarioAction{conn: conn}

	params, _ := json.Marshal(getScenarioParams{ScenarioID: 999})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "make.get_scenario",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for not found")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
}
