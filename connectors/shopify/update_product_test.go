package shopify

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestUpdateProduct_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if got := r.URL.Path; got != "/products/5001.json" {
			t.Errorf("path = %s, want /products/5001.json", got)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		product := reqBody["product"]
		if product["title"] != "Updated Widget" {
			t.Errorf("title = %v, want %q", product["title"], "Updated Widget")
		}
		if product["status"] != "active" {
			t.Errorf("status = %v, want %q", product["status"], "active")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"product": map[string]any{
				"id": 5001, "title": "Updated Widget", "status": "active",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.update_product"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.update_product",
		Parameters:  json.RawMessage(`{"product_id":5001,"title":"Updated Widget","status":"active"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if _, ok := data["product"]; !ok {
		t.Error("result missing 'product' key")
	}
}

func TestUpdateProduct_OnlyTagsProvided(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		product := reqBody["product"]
		if _, ok := product["title"]; ok {
			t.Error("title should not be present when not provided")
		}
		if product["tags"] != "sale,featured" {
			t.Errorf("tags = %v, want %q", product["tags"], "sale,featured")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"product": map[string]any{"id": 5001, "tags": "sale,featured"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.update_product"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.update_product",
		Parameters:  json.RawMessage(`{"product_id":5001,"tags":"sale,featured"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestUpdateProduct_WithVariants(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		product := reqBody["product"]
		variants, ok := product["variants"].([]interface{})
		if !ok {
			t.Fatal("variants not present or wrong type")
		}
		if len(variants) != 1 {
			t.Errorf("len(variants) = %d, want 1", len(variants))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"product": map[string]any{"id": 5001},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.update_product"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.update_product",
		Parameters:  json.RawMessage(`{"product_id":5001,"variants":[{"id":1,"price":"29.99"}]}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestUpdateProduct_InvalidProductID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.update_product"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.update_product",
		Parameters:  json.RawMessage(`{"product_id":0,"title":"Test"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdateProduct_NoFieldsProvided(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.update_product"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.update_product",
		Parameters:  json.RawMessage(`{"product_id":5001}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdateProduct_EmptyVariantsArray(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.update_product"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.update_product",
		Parameters:  json.RawMessage(`{"product_id":5001,"variants":[]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for empty variants array, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdateProduct_EmptyStatus(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.update_product"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.update_product",
		Parameters:  json.RawMessage(`{"product_id":5001,"status":""}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for empty status, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdateProduct_InvalidStatus(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.update_product"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.update_product",
		Parameters:  json.RawMessage(`{"product_id":5001,"status":"deleted"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdateProduct_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.update_product"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.update_product",
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

func TestUpdateProduct_APINotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errors":"Not Found"}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.update_product"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.update_product",
		Parameters:  json.RawMessage(`{"product_id":9999,"title":"Test"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 404, got %T: %v", err, err)
	}
}
