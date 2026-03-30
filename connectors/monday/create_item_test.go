package monday

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateItem_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "test-monday-api-key-123" {
			t.Errorf("expected API key in Authorization header, got %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("expected JSON content type, got %q", got)
		}

		var body graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if body.Query == "" {
			t.Error("expected non-empty query")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"create_item": map[string]any{
					"id":   "12345",
					"name": "New Task",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createItemAction{conn: conn}

	params, _ := json.Marshal(createItemParams{
		BoardID:  "9876",
		ItemName: "New Task",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.create_item",
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
	if data["id"] != "12345" {
		t.Errorf("expected id '12345', got %q", data["id"])
	}
	if data["name"] != "New Task" {
		t.Errorf("expected name 'New Task', got %q", data["name"])
	}
}

func TestCreateItem_WithColumnValues(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		// Verify column_values is passed as a string variable.
		cv, ok := body.Variables["column_values"]
		if !ok {
			t.Error("expected column_values in variables")
		}
		if _, ok := cv.(string); !ok {
			t.Errorf("expected column_values to be a string, got %T", cv)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"create_item": map[string]any{
					"id":   "12346",
					"name": "Task with cols",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createItemAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"board_id":  "9876",
		"item_name": "Task with cols",
		"column_values": map[string]any{
			"status": map[string]string{"label": "Working on it"},
		},
		"group_id": "topics",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.create_item",
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
	if data["id"] != "12346" {
		t.Errorf("expected id '12346', got %q", data["id"])
	}
}

func TestCreateItem_MissingBoardID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createItemAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"item_name": "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.create_item",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing board_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateItem_NonNumericBoardID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createItemAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"board_id":  "not-a-number",
		"item_name": "Test",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.create_item",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for non-numeric board_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateItem_MissingItemName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createItemAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"board_id": "9876",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.create_item",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing item_name")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateItem_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createItemAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.create_item",
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

func TestCreateItem_AuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error_code":    "UserUnauthorizedException",
			"error_message": "invalid token",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createItemAction{conn: conn}

	params, _ := json.Marshal(createItemParams{
		BoardID:  "9876",
		ItemName: "Test",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.create_item",
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

func TestCreateItem_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createItemAction{conn: conn}

	params, _ := json.Marshal(createItemParams{
		BoardID:  "9876",
		ItemName: "Test",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.create_item",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for rate limit")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
	var rlErr *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rlErr) {
		if rlErr.RetryAfter.Seconds() != 30 {
			t.Errorf("expected RetryAfter 30s, got %v", rlErr.RetryAfter)
		}
	}
}

func TestCreateItem_HTTP400_ValidationError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createItemAction{conn: conn}

	params, _ := json.Marshal(createItemParams{
		BoardID:  "9876",
		ItemName: "Test",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.create_item",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for HTTP 400")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for HTTP 400, got: %T", err)
	}
}

func TestCreateItem_GraphQLError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{
				{"message": "Board not found"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createItemAction{conn: conn}

	params, _ := json.Marshal(createItemParams{
		BoardID:  "9876",
		ItemName: "Test",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.create_item",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for GraphQL error")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
}
