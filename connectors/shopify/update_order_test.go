package shopify

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestUpdateOrder_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if got := r.URL.Path; got != "/orders/1001.json" {
			t.Errorf("path = %s, want /orders/1001.json", got)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		order := reqBody["order"]
		if order["note"] != "Updated by agent" {
			t.Errorf("note = %v, want %q", order["note"], "Updated by agent")
		}
		if order["tags"] != "vip,priority" {
			t.Errorf("tags = %v, want %q", order["tags"], "vip,priority")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"order": map[string]any{
				"id": 1001, "note": "Updated by agent", "tags": "vip,priority",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.update_order"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.update_order",
		Parameters:  json.RawMessage(`{"order_id":1001,"note":"Updated by agent","tags":"vip,priority"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if _, ok := data["order"]; !ok {
		t.Error("result missing 'order' key")
	}
}

func TestUpdateOrder_OnlyNote(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		order := reqBody["order"]
		if _, ok := order["tags"]; ok {
			t.Error("tags field should not be present when not provided")
		}
		if order["note"] != "Just a note" {
			t.Errorf("note = %v, want %q", order["note"], "Just a note")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"order": map[string]any{"id": 1001}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.update_order"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.update_order",
		Parameters:  json.RawMessage(`{"order_id":1001,"note":"Just a note"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestUpdateOrder_MissingOrderID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.update_order"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.update_order",
		Parameters:  json.RawMessage(`{"note":"test"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdateOrder_NoFieldsToUpdate(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.update_order"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.update_order",
		Parameters:  json.RawMessage(`{"order_id":1001}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdateOrder_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.update_order"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.update_order",
		Parameters:  json.RawMessage(`{invalid}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdateOrder_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"errors":{"email":["is invalid"]}}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.update_order"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.update_order",
		Parameters:  json.RawMessage(`{"order_id":1001,"email":"bad"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
