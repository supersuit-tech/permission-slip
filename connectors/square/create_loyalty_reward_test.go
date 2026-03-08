package square

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateLoyaltyReward_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/loyalty/rewards" {
			t.Errorf("path = %s, want /loyalty/rewards", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}
		if _, ok := reqBody["idempotency_key"]; !ok {
			t.Error("missing idempotency_key in request body")
		}
		if _, ok := reqBody["reward"]; !ok {
			t.Error("missing reward in request body")
		}

		var reward map[string]string
		json.Unmarshal(reqBody["reward"], &reward)
		if reward["loyalty_account_id"] != "LA123" {
			t.Errorf("loyalty_account_id = %q, want LA123", reward["loyalty_account_id"])
		}
		if reward["reward_tier_id"] != "TIER456" {
			t.Errorf("reward_tier_id = %q, want TIER456", reward["reward_tier_id"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"reward": map[string]any{
				"id":                 "RWD789",
				"loyalty_account_id": "LA123",
				"reward_tier_id":     "TIER456",
				"status":             "ISSUED",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.create_loyalty_reward"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.create_loyalty_reward",
		Parameters:  json.RawMessage(`{"loyalty_account_id": "LA123", "reward_tier_id": "TIER456"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if _, ok := data["reward"]; !ok {
		t.Error("result missing 'reward' key")
	}
}

func TestCreateLoyaltyReward_WithOrderID(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		json.Unmarshal(body, &reqBody)

		var reward map[string]string
		json.Unmarshal(reqBody["reward"], &reward)
		if reward["order_id"] != "ORDER001" {
			t.Errorf("order_id = %q, want ORDER001", reward["order_id"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"reward": map[string]any{"id": "RWD999"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.create_loyalty_reward"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.create_loyalty_reward",
		Parameters:  json.RawMessage(`{"loyalty_account_id": "LA123", "reward_tier_id": "TIER456", "order_id": "ORDER001"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreateLoyaltyReward_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["square.create_loyalty_reward"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing loyalty_account_id", `{"reward_tier_id": "TIER456"}`},
		{"empty loyalty_account_id", `{"loyalty_account_id": "", "reward_tier_id": "TIER456"}`},
		{"missing reward_tier_id", `{"loyalty_account_id": "LA123"}`},
		{"empty reward_tier_id", `{"loyalty_account_id": "LA123", "reward_tier_id": ""}`},
		{"invalid JSON", `{bad}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "square.create_loyalty_reward",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestCreateLoyaltyReward_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{
				{"category": "INVALID_REQUEST_ERROR", "code": "LOYALTY_ACCOUNT_NOT_FOUND", "detail": "Loyalty account not found."},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.create_loyalty_reward"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.create_loyalty_reward",
		Parameters:  json.RawMessage(`{"loyalty_account_id": "INVALID", "reward_tier_id": "TIER456"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
