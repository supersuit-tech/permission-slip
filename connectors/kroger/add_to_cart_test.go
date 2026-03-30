package kroger

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestAddToCart_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if got := r.URL.Path; got != "/cart/add" {
			t.Errorf("path = %s, want /cart/add", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test_access_token_123" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test_access_token_123")
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		items, ok := reqBody["items"].([]any)
		if !ok || len(items) != 2 {
			t.Errorf("expected 2 items, got %v", reqBody["items"])
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["kroger.add_to_cart"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.add_to_cart",
		Parameters:  json.RawMessage(`{"items":[{"upc":"0001111041700","quantity":2},{"upc":"0001111060903","quantity":1}]}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["status"] != "added" {
		t.Errorf("status = %v, want added", data["status"])
	}
	// items_count is a float64 when unmarshaled from JSON
	if data["items_count"] != float64(2) {
		t.Errorf("items_count = %v, want 2", data["items_count"])
	}
}

func TestAddToCart_EmptyItems(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["kroger.add_to_cart"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.add_to_cart",
		Parameters:  json.RawMessage(`{"items":[]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestAddToCart_MissingUPC(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["kroger.add_to_cart"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.add_to_cart",
		Parameters:  json.RawMessage(`{"items":[{"quantity":1}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestAddToCart_InvalidQuantity(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["kroger.add_to_cart"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.add_to_cart",
		Parameters:  json.RawMessage(`{"items":[{"upc":"0001111041700","quantity":0}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestAddToCart_TooManyItems(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["kroger.add_to_cart"]

	items := make([]cartItem, 26)
	for i := range items {
		items[i] = cartItem{UPC: "0001111041700", Quantity: 1}
	}
	params, _ := json.Marshal(map[string]any{"items": items})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.add_to_cart",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestAddToCart_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["kroger.add_to_cart"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.add_to_cart",
		Parameters:  json.RawMessage(`{invalid}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestAddToCart_AuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{{"message": "Unauthorized"}},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["kroger.add_to_cart"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.add_to_cart",
		Parameters:  json.RawMessage(`{"items":[{"upc":"0001111041700","quantity":1}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestAddToCart_MissingCredentials(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server with missing credentials")
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["kroger.add_to_cart"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.add_to_cart",
		Parameters:  json.RawMessage(`{"items":[{"upc":"0001111041700","quantity":1}]}`),
		Credentials: connectors.NewCredentials(map[string]string{}),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestAddToCart_WithModality(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if got, ok := reqBody["modality"].(string); !ok || got != "PICKUP" {
			t.Errorf("modality = %v, want PICKUP", reqBody["modality"])
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["kroger.add_to_cart"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.add_to_cart",
		Parameters:  json.RawMessage(`{"items":[{"upc":"0001111041700","quantity":1}],"modality":"PICKUP"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["status"] != "added" {
		t.Errorf("status = %v, want added", data["status"])
	}
}

func TestAddToCart_InvalidModality(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["kroger.add_to_cart"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.add_to_cart",
		Parameters:  json.RawMessage(`{"items":[{"upc":"0001111041700","quantity":1}],"modality":"INVALID"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestAddToCart_ModalityOmitted(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if _, ok := reqBody["modality"]; ok {
			t.Error("modality should not be present when omitted")
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["kroger.add_to_cart"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.add_to_cart",
		Parameters:  json.RawMessage(`{"items":[{"upc":"0001111041700","quantity":1}]}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}
