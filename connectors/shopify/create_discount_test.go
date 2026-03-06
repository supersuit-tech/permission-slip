package shopify

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateDiscount_Success(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := callCount.Add(1)

		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}

		body, _ := io.ReadAll(r.Body)

		if call == 1 {
			// Step 1: Create price rule.
			if got := r.URL.Path; got != "/price_rules.json" {
				t.Errorf("step 1 path = %s, want /price_rules.json", got)
			}

			var reqBody map[string]map[string]interface{}
			if err := json.Unmarshal(body, &reqBody); err != nil {
				t.Fatalf("unmarshal price rule body: %v", err)
			}
			pr := reqBody["price_rule"]
			if pr["title"] != "SUMMER10" {
				t.Errorf("title = %v, want %q", pr["title"], "SUMMER10")
			}
			if pr["value_type"] != "percentage" {
				t.Errorf("value_type = %v, want %q", pr["value_type"], "percentage")
			}
			if pr["value"] != "-10.0" {
				t.Errorf("value = %v, want %q", pr["value"], "-10.0")
			}
			if pr["target_type"] != "line_item" {
				t.Errorf("target_type = %v, want %q", pr["target_type"], "line_item")
			}

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{
				"price_rule": map[string]any{"id": 7001},
			})
		} else if call == 2 {
			// Step 2: Create discount code.
			if !strings.HasPrefix(r.URL.Path, "/price_rules/7001/discount_codes.json") {
				t.Errorf("step 2 path = %s, want /price_rules/7001/discount_codes.json", r.URL.Path)
			}

			var reqBody map[string]map[string]interface{}
			if err := json.Unmarshal(body, &reqBody); err != nil {
				t.Fatalf("unmarshal discount code body: %v", err)
			}
			dc := reqBody["discount_code"]
			if dc["code"] != "SUMMER10" {
				t.Errorf("code = %v, want %q", dc["code"], "SUMMER10")
			}

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{
				"discount_code": map[string]any{
					"id": 8001, "code": "SUMMER10", "price_rule_id": 7001,
				},
			})
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.create_discount"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_discount",
		Parameters:  json.RawMessage(`{"code":"SUMMER10","value_type":"percentage","value":"-10.0","starts_at":"2024-06-01T00:00:00Z"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if callCount.Load() != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount.Load())
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if data["price_rule_id"] != float64(7001) {
		t.Errorf("price_rule_id = %v, want 7001", data["price_rule_id"])
	}
	if _, ok := data["discount_code"]; !ok {
		t.Error("result missing 'discount_code' key")
	}
}

func TestCreateDiscount_WithAllOptions(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := callCount.Add(1)
		body, _ := io.ReadAll(r.Body)

		if call == 1 {
			var reqBody map[string]map[string]interface{}
			if err := json.Unmarshal(body, &reqBody); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			pr := reqBody["price_rule"]
			if pr["ends_at"] != "2024-07-01T00:00:00Z" {
				t.Errorf("ends_at = %v, want %q", pr["ends_at"], "2024-07-01T00:00:00Z")
			}
			if pr["usage_limit"] != float64(100) {
				t.Errorf("usage_limit = %v, want 100", pr["usage_limit"])
			}
			if pr["once_per_customer"] != true {
				t.Errorf("once_per_customer = %v, want true", pr["once_per_customer"])
			}
			if pr["target_type"] != "shipping_line" {
				t.Errorf("target_type = %v, want %q", pr["target_type"], "shipping_line")
			}

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{"price_rule": map[string]any{"id": 7002}})
		} else {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{
				"discount_code": map[string]any{"id": 8002, "code": "FREESHIP"},
			})
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.create_discount"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "shopify.create_discount",
		Parameters: json.RawMessage(`{
			"code": "FREESHIP",
			"value_type": "fixed_amount",
			"value": "-5.00",
			"target_type": "shipping_line",
			"starts_at": "2024-06-01T00:00:00Z",
			"ends_at": "2024-07-01T00:00:00Z",
			"usage_limit": 100,
			"applies_once_per_customer": true
		}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestCreateDiscount_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.create_discount"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing code", `{"value_type":"percentage","value":"-10","starts_at":"2024-01-01T00:00:00Z"}`},
		{"missing value_type", `{"code":"TEST","value":"-10","starts_at":"2024-01-01T00:00:00Z"}`},
		{"missing value", `{"code":"TEST","value_type":"percentage","starts_at":"2024-01-01T00:00:00Z"}`},
		{"missing starts_at", `{"code":"TEST","value_type":"percentage","value":"-10"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "shopify.create_discount",
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

func TestCreateDiscount_InvalidValueType(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.create_discount"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_discount",
		Parameters:  json.RawMessage(`{"code":"TEST","value_type":"invalid","value":"-10","starts_at":"2024-01-01T00:00:00Z"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateDiscount_InvalidTargetType(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.create_discount"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_discount",
		Parameters:  json.RawMessage(`{"code":"TEST","value_type":"percentage","value":"-10","target_type":"invalid","starts_at":"2024-01-01T00:00:00Z"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateDiscount_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.create_discount"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_discount",
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

func TestCreateDiscount_PriceRuleAPIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"errors":{"value":["must be negative"]}}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.create_discount"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_discount",
		Parameters:  json.RawMessage(`{"code":"TEST","value_type":"percentage","value":"10","starts_at":"2024-01-01T00:00:00Z"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateDiscount_DiscountCodeAPIError(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		call := callCount.Add(1)
		if call == 1 {
			// Price rule succeeds.
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{"price_rule": map[string]any{"id": 7003}})
		} else {
			// Discount code creation fails.
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(`{"errors":{"code":["has already been taken"]}}`))
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.create_discount"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_discount",
		Parameters:  json.RawMessage(`{"code":"DUPLICATE","value_type":"percentage","value":"-10","starts_at":"2024-01-01T00:00:00Z"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
