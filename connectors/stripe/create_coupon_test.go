package stripe

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateCoupon_PercentOff(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/coupons" {
			t.Errorf("path = %s, want /v1/coupons", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
			return
		}
		if got := r.FormValue("percent_off"); got != "25.5" {
			t.Errorf("percent_off = %q, want 25.5", got)
		}
		if got := r.FormValue("duration"); got != "once" {
			t.Errorf("duration = %q, want once", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":          "coupon_abc",
			"percent_off": 25.5,
			"duration":    "once",
			"valid":       true,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_coupon"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_coupon",
		Parameters:  json.RawMessage(`{"percent_off":25.5,"duration":"once"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != "coupon_abc" {
		t.Errorf("id = %v, want coupon_abc", data["id"])
	}
	if data["valid"] != true {
		t.Errorf("valid = %v, want true", data["valid"])
	}
}

func TestCreateCoupon_AmountOff(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
			return
		}
		if got := r.FormValue("amount_off"); got != "500" {
			t.Errorf("amount_off = %q, want 500", got)
		}
		if got := r.FormValue("currency"); got != "usd" {
			t.Errorf("currency = %q, want usd", got)
		}
		if got := r.FormValue("duration"); got != "repeating" {
			t.Errorf("duration = %q, want repeating", got)
		}
		if got := r.FormValue("duration_in_months"); got != "3" {
			t.Errorf("duration_in_months = %q, want 3", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":         "coupon_xyz",
			"amount_off": 500,
			"currency":   "usd",
			"duration":   "repeating",
			"valid":      true,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_coupon"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_coupon",
		Parameters:  json.RawMessage(`{"amount_off":500,"currency":"usd","duration":"repeating","duration_in_months":3}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != "coupon_xyz" {
		t.Errorf("id = %v, want coupon_xyz", data["id"])
	}
}

func TestCreateCoupon_MissingBothDiscountTypes(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_coupon"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_coupon",
		Parameters:  json.RawMessage(`{"duration":"once"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCoupon_BothDiscountTypes(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_coupon"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_coupon",
		Parameters:  json.RawMessage(`{"percent_off":25,"amount_off":500,"currency":"usd","duration":"once"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCoupon_MissingDuration(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_coupon"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_coupon",
		Parameters:  json.RawMessage(`{"percent_off":25}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCoupon_InvalidDuration(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_coupon"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_coupon",
		Parameters:  json.RawMessage(`{"percent_off":25,"duration":"bad"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCoupon_RepeatingWithoutMonths(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_coupon"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_coupon",
		Parameters:  json.RawMessage(`{"percent_off":25,"duration":"repeating"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCoupon_MonthsWithoutRepeating(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_coupon"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_coupon",
		Parameters:  json.RawMessage(`{"percent_off":25,"duration":"once","duration_in_months":3}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCoupon_AmountOffMissingCurrency(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_coupon"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_coupon",
		Parameters:  json.RawMessage(`{"amount_off":500,"duration":"once"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCoupon_PercentOffOutOfRange(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_coupon"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_coupon",
		Parameters:  json.RawMessage(`{"percent_off":150,"duration":"once"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCoupon_InvalidCurrency(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_coupon"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_coupon",
		Parameters:  json.RawMessage(`{"amount_off":500,"currency":"dollars","duration":"once"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
