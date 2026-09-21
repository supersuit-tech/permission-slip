package shopify

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func openFulfillmentOrder(id, locationID int64) map[string]any {
	return map[string]any{
		"id":                   id,
		"assigned_location_id": locationID,
		"status":               "open",
		"supported_actions":    []string{"create_fulfillment", "move", "hold"},
	}
}

func TestFulfillOrder_Success(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := callCount.Add(1)
		if call == 1 {
			if r.Method != http.MethodGet {
				t.Errorf("step 1 method = %s, want GET", r.Method)
			}
			if got := r.URL.Path; got != "/orders/1001/fulfillment_orders.json" {
				t.Errorf("step 1 path = %s, want /orders/1001/fulfillment_orders.json", got)
			}
			json.NewEncoder(w).Encode(map[string]any{
				"fulfillment_orders": []any{openFulfillmentOrder(5001, 10)},
			})
			return
		}

		if r.Method != http.MethodPost {
			t.Errorf("step 2 method = %s, want POST", r.Method)
		}
		if got := r.URL.Path; got != "/fulfillments.json" {
			t.Errorf("step 2 path = %s, want /fulfillments.json", got)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		fulfillment, ok := reqBody["fulfillment"].(map[string]interface{})
		if !ok {
			t.Fatal("missing fulfillment key in request body")
		}
		lineItems, ok := fulfillment["line_items_by_fulfillment_order"].([]interface{})
		if !ok || len(lineItems) != 1 {
			t.Fatalf("line_items_by_fulfillment_order = %v, want 1 item", fulfillment["line_items_by_fulfillment_order"])
		}
		item, ok := lineItems[0].(map[string]interface{})
		if !ok {
			t.Fatal("line_items_by_fulfillment_order[0] is not an object")
		}
		if item["fulfillment_order_id"] != float64(5001) {
			t.Errorf("fulfillment_order_id = %v, want 5001", item["fulfillment_order_id"])
		}
		trackingInfo, ok := fulfillment["tracking_info"].(map[string]interface{})
		if !ok {
			t.Fatal("missing tracking_info in fulfillment")
		}
		if trackingInfo["number"] != "1Z999AA10123456784" {
			t.Errorf("tracking number = %v, want 1Z999AA10123456784", trackingInfo["number"])
		}
		if trackingInfo["company"] != "UPS" {
			t.Errorf("tracking company = %v, want UPS", trackingInfo["company"])
		}
		if fulfillment["notify_customer"] != true {
			t.Errorf("notify_customer = %v, want true", fulfillment["notify_customer"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"fulfillment": map[string]any{
				"id": 3001, "order_id": 1001, "status": "success",
				"tracking_number": "1Z999AA10123456784", "tracking_company": "UPS",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.fulfill_order"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.fulfill_order",
		Parameters:  json.RawMessage(`{"order_id":1001,"tracking_number":"1Z999AA10123456784","tracking_company":"UPS","notify_customer":true}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if callCount.Load() != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount.Load())
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if _, ok := data["fulfillment"]; !ok {
		t.Error("result missing 'fulfillment' key")
	}
}

func TestFulfillOrder_MinimalParams(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := callCount.Add(1)
		if call == 1 {
			json.NewEncoder(w).Encode(map[string]any{
				"fulfillment_orders": []any{openFulfillmentOrder(5002, 10)},
			})
			return
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		fulfillment := reqBody["fulfillment"].(map[string]interface{})
		if _, ok := fulfillment["tracking_info"]; ok {
			t.Error("tracking_info should not be present when no tracking params provided")
		}
		lineItems := fulfillment["line_items_by_fulfillment_order"].([]interface{})
		if len(lineItems) != 1 {
			t.Errorf("line_items_by_fulfillment_order length = %d, want 1", len(lineItems))
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"fulfillment": map[string]any{"id": 3002, "order_id": 1002},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.fulfill_order"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.fulfill_order",
		Parameters:  json.RawMessage(`{"order_id":1002}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestFulfillOrder_SameLocationCombined(t *testing.T) {
	t.Parallel()

	var posted atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]any{
				"fulfillment_orders": []any{
					openFulfillmentOrder(5001, 10),
					openFulfillmentOrder(5002, 10),
					map[string]any{
						"id":                   5999,
						"assigned_location_id": 10,
						"status":               "closed",
						"supported_actions":    []string{},
					},
				},
			})
			return
		}

		posted.Add(1)
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		fulfillment := reqBody["fulfillment"].(map[string]interface{})
		lineItems := fulfillment["line_items_by_fulfillment_order"].([]interface{})
		if len(lineItems) != 2 {
			t.Errorf("line_items_by_fulfillment_order length = %d, want 2 (closed FO skipped)", len(lineItems))
		}
		ids := map[float64]bool{}
		for _, raw := range lineItems {
			item := raw.(map[string]interface{})
			id, ok := item["fulfillment_order_id"].(float64)
			if !ok {
				t.Fatalf("fulfillment_order_id is not a number: %v", item["fulfillment_order_id"])
			}
			ids[id] = true
		}
		if !ids[5001] || !ids[5002] {
			t.Errorf("fulfillment_order_ids = %v, want 5001 and 5002", ids)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"fulfillment": map[string]any{"id": 3003, "order_id": 1003},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.fulfill_order"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.fulfill_order",
		Parameters:  json.RawMessage(`{"order_id":1003}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if posted.Load() != 1 {
		t.Errorf("expected 1 create call, got %d", posted.Load())
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if _, ok := data["fulfillment"]; !ok {
		t.Error("result missing 'fulfillment' key for single combined fulfillment")
	}
}

func TestFulfillOrder_MultipleLocations(t *testing.T) {
	t.Parallel()

	var posted atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]any{
				"fulfillment_orders": []any{
					openFulfillmentOrder(5001, 10),
					openFulfillmentOrder(5002, 20),
				},
			})
			return
		}

		n := posted.Add(1)
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		fulfillment := reqBody["fulfillment"].(map[string]interface{})
		lineItems := fulfillment["line_items_by_fulfillment_order"].([]interface{})
		if len(lineItems) != 1 {
			t.Errorf("call %d: line_items_by_fulfillment_order length = %d, want 1 (one FO per location)", n, len(lineItems))
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"fulfillment": map[string]any{"id": 3000 + n, "order_id": 1004},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.fulfill_order"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.fulfill_order",
		Parameters:  json.RawMessage(`{"order_id":1004}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if posted.Load() != 2 {
		t.Errorf("expected 2 create calls, got %d", posted.Load())
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if _, ok := data["fulfillments"]; !ok {
		t.Error("result missing 'fulfillments' key for multi-location order")
	}
	if _, ok := data["fulfillment"]; ok {
		t.Error("single 'fulfillment' key should be omitted when multiple fulfillments are created")
	}
}

func TestFulfillOrder_NoFulfillableOrders(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"fulfillment_orders": []any{
				map[string]any{
					"id":                   5999,
					"assigned_location_id": 10,
					"status":               "closed",
					"supported_actions":    []string{},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.fulfill_order"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.fulfill_order",
		Parameters:  json.RawMessage(`{"order_id":1005}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "no fulfillable fulfillment orders") {
		t.Errorf("error should mention no fulfillable fulfillment orders, got: %v", err)
	}
}

func TestFulfillOrder_PartialCreateFailure(t *testing.T) {
	t.Parallel()

	var posted atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]any{
				"fulfillment_orders": []any{
					openFulfillmentOrder(5001, 10),
					openFulfillmentOrder(5002, 20),
				},
			})
			return
		}

		n := posted.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{
				"fulfillment": map[string]any{"id": 3001, "order_id": 1006},
			})
			return
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"errors":{"fulfillment":["unable to fulfill"]}}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.fulfill_order"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.fulfill_order",
		Parameters:  json.RawMessage(`{"order_id":1006}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError (wrapped), got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "location 20") {
		t.Errorf("error should mention failed location, got: %v", err)
	}
}

func TestFulfillOrder_InvalidOrderID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.fulfill_order"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.fulfill_order",
		Parameters:  json.RawMessage(`{"order_id":0}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestFulfillOrder_TrackingURLWithoutNumber(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.fulfill_order"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.fulfill_order",
		Parameters:  json.RawMessage(`{"order_id":1001,"tracking_url":"https://track.example.com/123"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestFulfillOrder_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.fulfill_order"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.fulfill_order",
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

func TestFulfillOrder_APINotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errors":"Not Found"}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.fulfill_order"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.fulfill_order",
		Parameters:  json.RawMessage(`{"order_id":9999}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 404, got %T: %v", err, err)
	}
}

func TestFulfillOrder_APIValidationError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]any{
				"fulfillment_orders": []any{openFulfillmentOrder(5001, 10)},
			})
			return
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"errors":{"order":["is already fulfilled"]}}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.fulfill_order"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.fulfill_order",
		Parameters:  json.RawMessage(`{"order_id":1001}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 422, got %T: %v", err, err)
	}
}

func TestFulfillmentOrder_CanCreateFulfillment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fo   fulfillmentOrder
		want bool
	}{
		{
			name: "supported action create_fulfillment",
			fo:   fulfillmentOrder{Status: "open", SupportedActions: []string{"create_fulfillment", "hold"}},
			want: true,
		},
		{
			name: "on hold with no create action",
			fo:   fulfillmentOrder{Status: "on_hold", SupportedActions: []string{"release_hold"}},
			want: false,
		},
		{
			name: "empty actions open status",
			fo:   fulfillmentOrder{Status: "open"},
			want: true,
		},
		{
			name: "empty actions in_progress status",
			fo:   fulfillmentOrder{Status: "in_progress"},
			want: true,
		},
		{
			name: "empty actions closed status",
			fo:   fulfillmentOrder{Status: "closed"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fo.canCreateFulfillment(); got != tt.want {
				t.Errorf("canCreateFulfillment() = %v, want %v", got, tt.want)
			}
		})
	}
}
