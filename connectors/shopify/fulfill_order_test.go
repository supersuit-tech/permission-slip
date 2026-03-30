package shopify

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestFulfillOrder_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.URL.Path; got != "/orders/1001/fulfillments.json" {
			t.Errorf("path = %s, want /orders/1001/fulfillments.json", got)
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

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		fulfillment := reqBody["fulfillment"].(map[string]interface{})
		if _, ok := fulfillment["tracking_info"]; ok {
			t.Error("tracking_info should not be present when no tracking params provided")
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

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
