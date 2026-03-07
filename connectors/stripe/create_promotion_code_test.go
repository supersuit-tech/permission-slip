package stripe

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreatePromotionCode_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/promotion_codes" {
			t.Errorf("path = %s, want /v1/promotion_codes", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
			return
		}
		if got := r.FormValue("coupon"); got != "coupon_abc" {
			t.Errorf("coupon = %q, want coupon_abc", got)
		}
		if got := r.FormValue("code"); got != "SUMMER25" {
			t.Errorf("code = %q, want SUMMER25", got)
		}
		if got := r.FormValue("max_redemptions"); got != "100" {
			t.Errorf("max_redemptions = %q, want 100", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":              "promo_abc123",
			"code":            "SUMMER25",
			"coupon":          map[string]any{"id": "coupon_abc"},
			"active":          true,
			"max_redemptions": 100,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_promotion_code"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_promotion_code",
		Parameters:  json.RawMessage(`{"coupon":"coupon_abc","code":"SUMMER25","max_redemptions":100}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != "promo_abc123" {
		t.Errorf("id = %v, want promo_abc123", data["id"])
	}
	if data["code"] != "SUMMER25" {
		t.Errorf("code = %v, want SUMMER25", data["code"])
	}
	if data["active"] != true {
		t.Errorf("active = %v, want true", data["active"])
	}
	// Verify coupon is returned as a nested object with an ID.
	coupon, ok := data["coupon"].(map[string]any)
	if !ok {
		t.Fatal("coupon should be a nested object")
	}
	if coupon["id"] != "coupon_abc" {
		t.Errorf("coupon.id = %v, want coupon_abc", coupon["id"])
	}
}

func TestCreatePromotionCode_MinimalParams(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
			return
		}
		if got := r.FormValue("coupon"); got != "coupon_abc" {
			t.Errorf("coupon = %q, want coupon_abc", got)
		}
		// code should not be set when omitted.
		if got := r.FormValue("code"); got != "" {
			t.Errorf("code should be empty, got %q", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":     "promo_auto",
			"code":   "AUTO_GENERATED",
			"coupon": map[string]any{"id": "coupon_abc"},
			"active": true,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_promotion_code"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_promotion_code",
		Parameters:  json.RawMessage(`{"coupon":"coupon_abc"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreatePromotionCode_MissingCoupon(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_promotion_code"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_promotion_code",
		Parameters:  json.RawMessage(`{"code":"SUMMER25"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreatePromotionCode_InvalidMaxRedemptions(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_promotion_code"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_promotion_code",
		Parameters:  json.RawMessage(`{"coupon":"coupon_abc","max_redemptions":-1}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreatePromotionCode_InvalidExpiresAt(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_promotion_code"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_promotion_code",
		Parameters:  json.RawMessage(`{"coupon":"coupon_abc","expires_at":-100}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
