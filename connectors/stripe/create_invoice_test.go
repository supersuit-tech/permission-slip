package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateInvoice_FullFlow(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		switch {
		case n == 1 && r.URL.Path == "/v1/invoices" && r.Method == http.MethodPost:
			// Step 1: Create invoice.
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parsing form: %v", err)
			}
			if got := r.FormValue("customer"); got != "cus_abc123" {
				t.Errorf("customer = %q, want cus_abc123", got)
			}
			if got := r.FormValue("description"); got != "Monthly services" {
				t.Errorf("description = %q, want Monthly services", got)
			}
			if got := r.Header.Get("Idempotency-Key"); got == "" {
				t.Error("expected Idempotency-Key on POST")
			}

			json.NewEncoder(w).Encode(map[string]any{
				"id":     "in_test123",
				"status": "draft",
			})

		case n == 2 && r.URL.Path == "/v1/invoiceitems" && r.Method == http.MethodPost:
			// Step 2: Add line item.
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parsing form: %v", err)
			}
			if got := r.FormValue("invoice"); got != "in_test123" {
				t.Errorf("invoice = %q, want in_test123", got)
			}
			if got := r.FormValue("amount"); got != "5000" {
				t.Errorf("amount = %q, want 5000", got)
			}
			if got := r.FormValue("description"); got != "Consulting" {
				t.Errorf("description = %q, want Consulting", got)
			}

			json.NewEncoder(w).Encode(map[string]any{"id": "ii_001"})

		case n == 3 && r.URL.Path == "/v1/invoices/in_test123/finalize" && r.Method == http.MethodPost:
			// Step 3: Finalize.
			json.NewEncoder(w).Encode(map[string]any{
				"id":     "in_test123",
				"status": "open",
			})

		default:
			t.Errorf("unexpected request #%d: %s %s", n, r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_invoice"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "stripe.create_invoice",
		Parameters: json.RawMessage(`{
			"customer_id": "cus_abc123",
			"description": "Monthly services",
			"line_items": [{"description": "Consulting", "amount": 5000, "quantity": 1}]
		}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != "in_test123" {
		t.Errorf("id = %v, want in_test123", data["id"])
	}
	if data["status"] != "open" {
		t.Errorf("status = %v, want open (finalized)", data["status"])
	}
	if got := callCount.Load(); got != 3 {
		t.Errorf("expected 3 API calls (create + item + finalize), got %d", got)
	}
}

func TestCreateInvoice_NoFinalize(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		switch {
		case n == 1 && r.URL.Path == "/v1/invoices":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parsing form: %v", err)
			}
			if got := r.FormValue("auto_advance"); got != "false" {
				t.Errorf("auto_advance = %q, want false", got)
			}
			json.NewEncoder(w).Encode(map[string]any{
				"id":     "in_draft",
				"status": "draft",
			})
		default:
			t.Errorf("unexpected request #%d: %s %s", n, r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_invoice"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_invoice",
		Parameters:  json.RawMessage(`{"customer_id":"cus_abc123","auto_advance":false}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["status"] != "draft" {
		t.Errorf("status = %v, want draft (not finalized)", data["status"])
	}
	if got := callCount.Load(); got != 1 {
		t.Errorf("expected 1 API call (create only, no finalize), got %d", got)
	}
}

func TestCreateInvoice_MissingCustomerID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_invoice"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_invoice",
		Parameters:  json.RawMessage(`{"description":"Test"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateInvoice_InvalidLineItemAmount(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_invoice"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_invoice",
		Parameters:  json.RawMessage(`{"customer_id":"cus_abc123","line_items":[{"amount":0}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateInvoice_MultipleLineItems(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		switch {
		case n == 1 && r.URL.Path == "/v1/invoices":
			json.NewEncoder(w).Encode(map[string]any{"id": "in_multi", "status": "draft"})
		case (n == 2 || n == 3) && r.URL.Path == "/v1/invoiceitems":
			json.NewEncoder(w).Encode(map[string]any{"id": fmt.Sprintf("ii_%d", n)})
		case n == 4 && r.URL.Path == "/v1/invoices/in_multi/finalize":
			json.NewEncoder(w).Encode(map[string]any{"id": "in_multi", "status": "open"})
		default:
			t.Errorf("unexpected request #%d: %s %s", n, r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_invoice"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "stripe.create_invoice",
		Parameters: json.RawMessage(`{
			"customer_id": "cus_abc123",
			"line_items": [
				{"description": "Item A", "amount": 1000, "quantity": 1},
				{"description": "Item B", "amount": 2000, "quantity": 2}
			]
		}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if got := callCount.Load(); got != 4 {
		t.Errorf("expected 4 API calls (create + 2 items + finalize), got %d", got)
	}
}

func TestCreateInvoice_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_invoice"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  "stripe.create_invoice",
		Parameters:  json.RawMessage(`{"customer_id":"cus_abc123"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}
