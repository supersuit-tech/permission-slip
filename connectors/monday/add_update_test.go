package monday

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestAddUpdate_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"create_update": map[string]any{
					"id":   "99001",
					"body": "This is a comment",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &addUpdateAction{conn: conn}

	params, _ := json.Marshal(addUpdateParams{
		ItemID: "12345",
		Body:   "This is a comment",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.add_update",
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
	if data["id"] != "99001" {
		t.Errorf("expected id '99001', got %q", data["id"])
	}
	if data["body"] != "This is a comment" {
		t.Errorf("expected body 'This is a comment', got %q", data["body"])
	}
}

func TestAddUpdate_MissingItemID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addUpdateAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"body": "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.add_update",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing item_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAddUpdate_NonNumericItemID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addUpdateAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"item_id": "abc-def",
		"body":    "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.add_update",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for non-numeric item_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAddUpdate_MissingBody(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addUpdateAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"item_id": "12345",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.add_update",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing body")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAddUpdate_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addUpdateAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.add_update",
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
