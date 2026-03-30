package make

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListScenarios_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Token test-api-token-123" {
			t.Errorf("expected Token auth header, got %q", got)
		}
		if got := r.URL.Query().Get("teamId"); got != "42" {
			t.Errorf("expected teamId=42, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"scenarios": []map[string]any{
				{"id": 1, "name": "Test Scenario", "isEnabled": true},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listScenariosAction{conn: conn}

	params, _ := json.Marshal(listScenariosParams{TeamID: 42})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "make.list_scenarios",
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
	if data["scenarios"] == nil {
		t.Error("expected scenarios in response")
	}
}

func TestListScenarios_InvalidTeamID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listScenariosAction{conn: conn}

	params, _ := json.Marshal(listScenariosParams{TeamID: 0})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "make.list_scenarios",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid team_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListScenarios_AuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Unauthorized"}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listScenariosAction{conn: conn}

	params, _ := json.Marshal(listScenariosParams{TeamID: 42})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "make.list_scenarios",
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

func TestListScenarios_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listScenariosAction{conn: conn}

	params, _ := json.Marshal(listScenariosParams{TeamID: 42})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "make.list_scenarios",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for rate limit")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
}
