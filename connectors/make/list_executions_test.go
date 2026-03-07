package make

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListExecutions_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/scenarios/42/logs" {
			t.Errorf("expected path /scenarios/42/logs, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"scenarioLogs": []map[string]any{
				{"id": "log-1", "status": "success"},
				{"id": "log-2", "status": "error"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listExecutionsAction{conn: conn}

	params, _ := json.Marshal(listExecutionsParams{ScenarioID: 42})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "make.list_executions",
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
	if data["scenarioLogs"] == nil {
		t.Error("expected scenarioLogs in response")
	}
}

func TestListExecutions_InvalidScenarioID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listExecutionsAction{conn: conn}

	params, _ := json.Marshal(listExecutionsParams{ScenarioID: -1})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "make.list_executions",
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

func TestListExecutions_InvalidLimit(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listExecutionsAction{conn: conn}

	params, _ := json.Marshal(listExecutionsParams{ScenarioID: 42, Limit: 200})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "make.list_executions",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid limit")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListExecutions_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listExecutionsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "make.list_executions",
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
