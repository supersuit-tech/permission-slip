package stripe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListInvoices_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/invoices" {
			t.Errorf("path = %s, want /v1/invoices", r.URL.Path)
		}
		if got := r.URL.Query().Get("customer"); got != "cus_abc123" {
			t.Errorf("customer = %q, want cus_abc123", got)
		}
		if got := r.URL.Query().Get("status"); got != "paid" {
			t.Errorf("status = %q, want paid", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"object":   "list",
			"data":     []any{},
			"has_more": false,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.list_invoices"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.list_invoices",
		Parameters:  json.RawMessage(`{"customer_id":"cus_abc123","status":"paid"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["has_more"] != false {
		t.Errorf("has_more = %v, want false", data["has_more"])
	}
}

func TestListInvoices_InvalidStatus(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.list_invoices"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.list_invoices",
		Parameters:  json.RawMessage(`{"status":"invalid"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestListInvoices_DefaultLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Errorf("limit = %q, want 10 (default)", got)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"object":   "list",
			"data":     []any{},
			"has_more": false,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.list_invoices"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.list_invoices",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestListInvoices_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	_, err := conn.Actions()["stripe.list_invoices"].Execute(ctx, connectors.ActionRequest{
		ActionType:  "stripe.list_invoices",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}
