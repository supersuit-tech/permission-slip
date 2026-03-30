package make

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestRunScenario_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/scenarios/123/run" {
			t.Errorf("expected path /scenarios/123/run, got %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"executionId": "exec-abc-123",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &runScenarioAction{conn: conn}

	params, _ := json.Marshal(runScenarioParams{ScenarioID: 123})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "make.run_scenario",
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
	if data["executionId"] != "exec-abc-123" {
		t.Errorf("expected executionId 'exec-abc-123', got %v", data["executionId"])
	}
}

func TestRunScenario_WithData(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		data, ok := body["data"].(map[string]any)
		if !ok {
			t.Errorf("expected data in request body")
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if data["input_field"] != "test_value" {
			t.Errorf("expected input_field 'test_value', got %v", data["input_field"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"executionId": "exec-456"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &runScenarioAction{conn: conn}

	inputData := json.RawMessage(`{"input_field":"test_value"}`)
	params, _ := json.Marshal(runScenarioParams{
		ScenarioID: 123,
		Data:       &inputData,
		Responsive: true,
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "make.run_scenario",
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

func TestRunScenario_InvalidScenarioID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &runScenarioAction{conn: conn}

	params, _ := json.Marshal(runScenarioParams{ScenarioID: 0})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "make.run_scenario",
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

func TestRunScenario_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"Internal error"}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &runScenarioAction{conn: conn}

	params, _ := json.Marshal(runScenarioParams{ScenarioID: 123})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "make.run_scenario",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for server error")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
}
