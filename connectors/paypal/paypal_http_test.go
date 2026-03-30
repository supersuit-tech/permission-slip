package paypal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCheckResponse_Success(t *testing.T) {
	t.Parallel()
	if err := checkResponse(200, nil, []byte(`{}`)); err != nil {
		t.Errorf("checkResponse(200): %v", err)
	}
}

func TestCheckResponse_Auth(t *testing.T) {
	t.Parallel()
	body := `{"name":"AUTHENTICATION_FAILURE","message":"Invalid access token","debug_id":"abc"}`
	err := checkResponse(401, nil, []byte(body))
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("want AuthError, got %T: %v", err, err)
	}
}

func TestCheckResponse_Validation(t *testing.T) {
	t.Parallel()
	body := `{"name":"INVALID_REQUEST","message":"Malformed request","debug_id":"x"}`
	err := checkResponse(400, nil, []byte(body))
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("want ValidationError, got %T: %v", err, err)
	}
}

func TestCheckResponse_RateLimit(t *testing.T) {
	t.Parallel()
	h := http.Header{}
	h.Set("Retry-After", "7")
	err := checkResponse(429, h, []byte(`{"message":"throttled"}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("want RateLimitError, got %T: %v", err, err)
	}
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) && rle.RetryAfter != 7*time.Second {
		t.Errorf("RetryAfter = %v, want 7s", rle.RetryAfter)
	}
}

func TestValidatePayPalPathID(t *testing.T) {
	t.Parallel()
	if err := validatePayPalPathID("id", ""); err == nil {
		t.Error("empty id: want error")
	}
	if err := validatePayPalPathID("id", "ab/cd"); err == nil {
		t.Error("slash: want error")
	}
	if err := validatePayPalPathID("id", "PAYOUT-123"); err != nil {
		t.Errorf("valid id: %v", err)
	}
}

func TestReadJSONBody_TooLarge(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(make([]byte, maxJSONBodyBytes+1))
	if _, err := readJSONBody(raw, "x"); err == nil {
		t.Fatal("expected error for oversized body")
	}
}

func TestCreateOrderAction_HTTPServer(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/checkout/orders" || r.Method != http.MethodPost {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("auth %q", r.Header.Get("Authorization"))
		}
		rid := r.Header.Get("PayPal-Request-Id")
		if rid == "" {
			t.Error("missing PayPal-Request-Id header")
		}
		if len(rid) > maxRequestIDLen {
			t.Errorf("PayPal-Request-Id length = %d, want <= %d", len(rid), maxRequestIDLen)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"ORD-1","status":"CREATED"}`))
	}))
	t.Cleanup(srv.Close)

	c := newForTest(srv.Client(), srv.URL)
	act := &createOrderAction{conn: c}
	params := map[string]any{
		"order": map[string]any{"intent": "CAPTURE"},
	}
	raw, _ := json.Marshal(params)
	res, err := act.Execute(context.Background(), connectors.ActionRequest{
		ActionType:  "paypal.create_order",
		Parameters:  raw,
		Credentials: connectors.NewCredentials(map[string]string{"access_token": "tok"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(res.Data) {
		t.Fatalf("invalid json: %s", res.Data)
	}
}

func TestGetOrderAction_HTTPServer(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/checkout/orders/ORD-9" || r.Method != http.MethodGet {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"ORD-9","status":"APPROVED"}`))
	}))
	t.Cleanup(srv.Close)

	c := newForTest(srv.Client(), srv.URL)
	act := &getOrderAction{conn: c}
	raw, _ := json.Marshal(map[string]string{"order_id": "ORD-9"})
	res, err := act.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  raw,
		Credentials: connectors.NewCredentials(map[string]string{"access_token": "tok"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(res.Data, &out); err != nil {
		t.Fatal(err)
	}
	if out["id"] != "ORD-9" {
		t.Errorf("id = %v", out["id"])
	}
}

func TestHTTPClient_DoesNotFollowRedirects(t *testing.T) {
	t.Parallel()
	var handlerCalls int
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalls++
		if r.URL.Path == "/v2/checkout/orders" && r.Method == http.MethodPost {
			http.Redirect(w, r, ts.URL+"/trap", http.StatusFound)
			return
		}
		// If the client followed the redirect, it would POST here with the Bearer token.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"trapped":true}`))
	}))
	t.Cleanup(ts.Close)
	srv := ts

	c := newForTest(New().client, srv.URL)
	act := &createOrderAction{conn: c}
	raw, _ := json.Marshal(map[string]any{"order": map[string]any{"intent": "CAPTURE"}})
	_, err := act.Execute(context.Background(), connectors.ActionRequest{
		ActionType:  "paypal.create_order",
		Parameters:  raw,
		Credentials: connectors.NewCredentials(map[string]string{"access_token": "secret-token"}),
	})
	if err == nil {
		t.Fatal("expected error on redirect response")
	}
	if handlerCalls != 1 {
		t.Fatalf("handler called %d times, want 1 (redirect must not be followed)", handlerCalls)
	}
}

func TestValidateCredentials(t *testing.T) {
	t.Parallel()
	p := New()
	if err := p.ValidateCredentials(context.Background(), connectors.NewCredentials(nil)); err == nil {
		t.Error("missing token: want error")
	}
	if err := p.ValidateCredentials(context.Background(), connectors.NewCredentials(map[string]string{
		"access_token": "x",
		"environment":  "prod",
	})); err == nil {
		t.Error("bad environment: want error")
	}
	if err := p.ValidateCredentials(context.Background(), connectors.NewCredentials(map[string]string{
		"access_token": "x",
	})); err != nil {
		t.Errorf("valid: %v", err)
	}
	// Whitespace around environment should be trimmed (treated as sandbox).
	if err := p.ValidateCredentials(context.Background(), connectors.NewCredentials(map[string]string{
		"access_token": "x",
		"environment":  "  sandbox  ",
	})); err != nil {
		t.Errorf("trimmed sandbox env: %v", err)
	}
}

func TestDeriveRequestID_FitsPayPalLimit(t *testing.T) {
	t.Parallel()
	// Use a realistic action type and large params to ensure SHA256 output is truncated.
	params := json.RawMessage(`{"order":{"intent":"CAPTURE","purchase_units":[{"amount":{"currency_code":"USD","value":"99.99"}}]}}`)
	id := deriveRequestID("paypal.create_order", params)
	if len(id) > maxRequestIDLen {
		t.Errorf("deriveRequestID length = %d, want <= %d; value = %q", len(id), maxRequestIDLen, id)
	}
	if len(id) == 0 {
		t.Error("deriveRequestID returned empty string")
	}
	// Same inputs must produce same output (deterministic).
	id2 := deriveRequestID("paypal.create_order", params)
	if id != id2 {
		t.Errorf("deriveRequestID not deterministic: %q != %q", id, id2)
	}
	// Different inputs must produce different output.
	id3 := deriveRequestID("paypal.capture_order", params)
	if id == id3 {
		t.Error("different action types produced same request ID")
	}
}

func TestAPIBaseURLForCreds_SandboxTrimmed(t *testing.T) {
	t.Parallel()
	c := connectors.NewCredentials(map[string]string{
		"access_token": "x",
		"environment":  "\tsandbox\n",
	})
	if apiBaseURLForCreds(c) != sandboxAPIBaseURL {
		t.Errorf("got %q, want sandbox", apiBaseURLForCreds(c))
	}
}
