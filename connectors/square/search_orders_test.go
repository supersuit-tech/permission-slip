package square

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSearchOrders_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/orders/search" {
			t.Errorf("path = %s, want /orders/search", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}

		var locationIDs []string
		json.Unmarshal(reqBody["location_ids"], &locationIDs)
		if len(locationIDs) != 1 || locationIDs[0] != "L123" {
			t.Errorf("location_ids = %v, want [L123]", locationIDs)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"orders": []map[string]any{
				{"id": "ORD1", "state": "OPEN"},
				{"id": "ORD2", "state": "COMPLETED"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.search_orders"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.search_orders",
		Parameters:  json.RawMessage(`{"location_ids": ["L123"]}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if _, ok := data["orders"]; !ok {
		t.Error("result missing 'orders' key")
	}
}

func TestSearchOrders_WithQueryAndLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		json.Unmarshal(body, &reqBody)

		if _, ok := reqBody["query"]; !ok {
			t.Error("missing query in request body")
		}

		var limit int
		json.Unmarshal(reqBody["limit"], &limit)
		if limit != 10 {
			t.Errorf("limit = %d, want 10", limit)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"orders": []any{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.search_orders"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "square.search_orders",
		Parameters: json.RawMessage(`{
			"location_ids": ["L123"],
			"query": {"filter": {"state_filter": {"states": ["OPEN"]}}},
			"limit": 10
		}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSearchOrders_WithPagination(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		json.Unmarshal(body, &reqBody)

		resp := map[string]any{"orders": []any{}}
		if _, ok := reqBody["cursor"]; !ok {
			resp["cursor"] = "page2"
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.search_orders"]

	// First page.
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.search_orders",
		Parameters:  json.RawMessage(`{"location_ids": ["L123"]}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	json.Unmarshal(result.Data, &data)
	if data["cursor"] != "page2" {
		t.Errorf("cursor = %v, want page2", data["cursor"])
	}

	// Second page.
	result, err = action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.search_orders",
		Parameters:  json.RawMessage(`{"location_ids": ["L123"], "cursor": "page2"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data2 map[string]any
	json.Unmarshal(result.Data, &data2)
	if _, ok := data2["cursor"]; ok {
		t.Error("cursor should not be present on last page")
	}
}

func TestSearchOrders_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["square.search_orders"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing location_ids", params: `{}`},
		{name: "empty location_ids", params: `{"location_ids": []}`},
		{name: "query not an object", params: `{"location_ids": ["L123"], "query": "OPEN"}`},
		{name: "query is array", params: `{"location_ids": ["L123"], "query": [1,2,3]}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "square.search_orders",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("Execute() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestSearchOrders_NullOrdersNormalized(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"orders": null}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.search_orders"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.search_orders",
		Parameters:  json.RawMessage(`{"location_ids": ["L123"]}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]json.RawMessage
	json.Unmarshal(result.Data, &data)
	if string(data["orders"]) != "[]" {
		t.Errorf("orders = %s, want [] (null should be normalized to empty array)", string(data["orders"]))
	}
}

func TestSearchOrders_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{
				{"category": "API_ERROR", "code": "INTERNAL_SERVER_ERROR", "detail": "Unexpected error"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.search_orders"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.search_orders",
		Parameters:  json.RawMessage(`{"location_ids": ["L123"]}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestSearchOrders_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.search_orders"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  "square.search_orders",
		Parameters:  json.RawMessage(`{"location_ids": ["L123"]}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}
