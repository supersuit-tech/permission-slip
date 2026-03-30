package shopify

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateProduct_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.URL.Path; got != "/products.json" {
			t.Errorf("path = %s, want /products.json", got)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		product := reqBody["product"]
		if product["title"] != "Test Widget" {
			t.Errorf("title = %v, want %q", product["title"], "Test Widget")
		}
		if product["vendor"] != "TestCo" {
			t.Errorf("vendor = %v, want %q", product["vendor"], "TestCo")
		}
		if product["status"] != "draft" {
			t.Errorf("status = %v, want %q", product["status"], "draft")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"product": map[string]any{
				"id": 5001, "title": "Test Widget", "vendor": "TestCo",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.create_product"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_product",
		Parameters:  json.RawMessage(`{"title":"Test Widget","vendor":"TestCo","status":"draft"}`),
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

func TestCreateProduct_WithVariants(t *testing.T) {
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
		if len(variants) != 2 {
			t.Errorf("len(variants) = %d, want 2", len(variants))
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"product": map[string]any{"id": 5002}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.create_product"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "shopify.create_product",
		Parameters: json.RawMessage(`{
			"title": "T-Shirt",
			"variants": [
				{"price": "19.99", "sku": "TS-SM", "option1": "Small"},
				{"price": "19.99", "sku": "TS-LG", "option1": "Large"}
			]
		}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestCreateProduct_TitleOnly(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		product := reqBody["product"]
		// Only title should be present.
		if _, ok := product["vendor"]; ok {
			t.Error("vendor should not be present when empty")
		}
		if _, ok := product["body_html"]; ok {
			t.Error("body_html should not be present when empty")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"product": map[string]any{"id": 5003}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.create_product"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_product",
		Parameters:  json.RawMessage(`{"title":"Minimal Product"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestCreateProduct_MissingTitle(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.create_product"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_product",
		Parameters:  json.RawMessage(`{"vendor":"TestCo"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateProduct_InvalidStatus(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.create_product"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_product",
		Parameters:  json.RawMessage(`{"title":"Widget","status":"deleted"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateProduct_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.create_product"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_product",
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

func TestCreateProduct_APIValidationError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"errors":{"title":["can't be blank"]}}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.create_product"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_product",
		Parameters:  json.RawMessage(`{"title":"Widget"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
