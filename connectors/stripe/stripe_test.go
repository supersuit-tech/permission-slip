package stripe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// ---------------------------------------------------------------------------
// ValidateCredentials
// ---------------------------------------------------------------------------

func TestValidateCredentials_Valid(t *testing.T) {
	t.Parallel()

	conn := New()
	tests := []struct {
		name string
		key  string
	}{
		{"live key", "sk_live_abc123"},
		{"test key", "sk_test_abc123"},
		{"restricted live key", "rk_live_abc123"},
		{"restricted test key", "rk_test_abc123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds := connectors.NewCredentials(map[string]string{"api_key": tt.key})
			if err := conn.ValidateCredentials(t.Context(), creds); err != nil {
				t.Errorf("ValidateCredentials() unexpected error: %v", err)
			}
		})
	}
}

func TestValidateCredentials_Invalid(t *testing.T) {
	t.Parallel()

	conn := New()
	tests := []struct {
		name  string
		creds connectors.Credentials
	}{
		{"missing key", connectors.NewCredentials(map[string]string{})},
		{"empty key", connectors.NewCredentials(map[string]string{"api_key": ""})},
		{"bad prefix", connectors.NewCredentials(map[string]string{"api_key": "pk_test_abc123"})},
		{"wrong cred name", connectors.NewCredentials(map[string]string{"token": "sk_test_abc123"})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := conn.ValidateCredentials(t.Context(), tt.creds)
			if err == nil {
				t.Fatal("ValidateCredentials() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// formEncode
// ---------------------------------------------------------------------------

func TestFormEncode_FlatValues(t *testing.T) {
	t.Parallel()

	result := formEncode(map[string]any{
		"email": "test@example.com",
		"name":  "Test User",
	})

	if result["email"] != "test@example.com" {
		t.Errorf("email = %q, want %q", result["email"], "test@example.com")
	}
	if result["name"] != "Test User" {
		t.Errorf("name = %q, want %q", result["name"], "Test User")
	}
}

func TestFormEncode_NestedObject(t *testing.T) {
	t.Parallel()

	result := formEncode(map[string]any{
		"metadata": map[string]any{
			"order_id": "12345",
			"source":   "agent",
		},
	})

	if result["metadata[order_id]"] != "12345" {
		t.Errorf("metadata[order_id] = %q, want %q", result["metadata[order_id]"], "12345")
	}
	if result["metadata[source]"] != "agent" {
		t.Errorf("metadata[source] = %q, want %q", result["metadata[source]"], "agent")
	}
}

func TestFormEncode_Array(t *testing.T) {
	t.Parallel()

	result := formEncode(map[string]any{
		"line_items": []any{
			map[string]any{
				"price":    "price_abc",
				"quantity": float64(2),
			},
			map[string]any{
				"price":    "price_def",
				"quantity": float64(1),
			},
		},
	})

	if result["line_items[0][price]"] != "price_abc" {
		t.Errorf("line_items[0][price] = %q, want %q", result["line_items[0][price]"], "price_abc")
	}
	if result["line_items[0][quantity]"] != "2" {
		t.Errorf("line_items[0][quantity] = %q, want %q", result["line_items[0][quantity]"], "2")
	}
	if result["line_items[1][price]"] != "price_def" {
		t.Errorf("line_items[1][price] = %q, want %q", result["line_items[1][price]"], "price_def")
	}
}

func TestFormEncode_NilSkipped(t *testing.T) {
	t.Parallel()

	result := formEncode(map[string]any{
		"email":       "test@example.com",
		"description": nil,
	})

	if _, ok := result["description"]; ok {
		t.Error("nil value should be skipped, but description key is present")
	}
	if len(result) != 1 {
		t.Errorf("expected 1 key, got %d", len(result))
	}
}

func TestFormEncode_BooleanAndNumber(t *testing.T) {
	t.Parallel()

	result := formEncode(map[string]any{
		"auto_advance": true,
		"amount":       float64(1500),
	})

	if result["auto_advance"] != "true" {
		t.Errorf("auto_advance = %q, want %q", result["auto_advance"], "true")
	}
	if result["amount"] != "1500" {
		t.Errorf("amount = %q, want %q", result["amount"], "1500")
	}
}

func TestFormEncode_Empty(t *testing.T) {
	t.Parallel()

	result := formEncode(map[string]any{})
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d keys", len(result))
	}
}

// ---------------------------------------------------------------------------
// encodeParams (deterministic ordering)
// ---------------------------------------------------------------------------

func TestEncodeParams_Sorted(t *testing.T) {
	t.Parallel()

	encoded := encodeParams(map[string]string{
		"email": "a@b.com",
		"name":  "Test",
		"amount": "100",
	})

	// url.Values.Encode sorts keys alphabetically.
	want := "amount=100&email=a%40b.com&name=Test"
	if encoded != want {
		t.Errorf("encodeParams = %q, want %q", encoded, want)
	}
}

// ---------------------------------------------------------------------------
// checkResponse (error mapping)
// ---------------------------------------------------------------------------

func TestCheckResponse_Success(t *testing.T) {
	t.Parallel()

	if err := checkResponse(200, http.Header{}, []byte(`{}`)); err != nil {
		t.Errorf("checkResponse(200) unexpected error: %v", err)
	}
}

func TestCheckResponse_AuthenticationError(t *testing.T) {
	t.Parallel()

	body := `{"error":{"type":"authentication_error","message":"Invalid API Key provided"}}`
	err := checkResponse(401, http.Header{}, []byte(body))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestCheckResponse_InvalidRequestError(t *testing.T) {
	t.Parallel()

	body := `{"error":{"type":"invalid_request_error","message":"Missing required param: customer"}}`
	err := checkResponse(400, http.Header{}, []byte(body))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCheckResponse_RateLimitHTTP429(t *testing.T) {
	t.Parallel()

	header := http.Header{}
	header.Set("Retry-After", "5")
	body := `{"error":{"type":"rate_limit_error","message":"Too many requests"}}`
	err := checkResponse(429, header, []byte(body))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) {
		if rle.RetryAfter != 5*time.Second {
			t.Errorf("RetryAfter = %v, want 5s", rle.RetryAfter)
		}
	}
}

func TestCheckResponse_RateLimitErrorType(t *testing.T) {
	t.Parallel()

	// Stripe can return rate_limit_error type on non-429 status codes.
	body := `{"error":{"type":"rate_limit_error","message":"Too many requests"}}`
	err := checkResponse(400, http.Header{}, []byte(body))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
}

func TestCheckResponse_CardError(t *testing.T) {
	t.Parallel()

	body := `{"error":{"type":"card_error","code":"card_declined","message":"Your card was declined"}}`
	err := checkResponse(402, http.Header{}, []byte(body))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestCheckResponse_APIError(t *testing.T) {
	t.Parallel()

	body := `{"error":{"type":"api_error","message":"Internal error"}}`
	err := checkResponse(500, http.Header{}, []byte(body))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestCheckResponse_UnknownErrorFallback(t *testing.T) {
	t.Parallel()

	err := checkResponse(503, http.Header{}, []byte(`Service Unavailable`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestCheckResponse_401WithoutStripeBody(t *testing.T) {
	t.Parallel()

	err := checkResponse(401, http.Header{}, []byte(`Unauthorized`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

// ---------------------------------------------------------------------------
// do() integration tests via httptest
// ---------------------------------------------------------------------------

func TestDo_GET_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk_test_abc123" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer sk_test_abc123")
		}
		if r.URL.Query().Get("customer") != "cus_123" {
			t.Errorf("customer param = %q, want %q", r.URL.Query().Get("customer"), "cus_123")
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{"id": "sub_123"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/v1/subscriptions", map[string]string{"customer": "cus_123"}, &resp, "")
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
	if resp["id"] != "sub_123" {
		t.Errorf("id = %v, want sub_123", resp["id"])
	}
}

func TestDo_POST_FormEncoded(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", ct)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if r.Form.Get("email") != "test@example.com" {
			t.Errorf("email = %q, want test@example.com", r.Form.Get("email"))
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"id": "cus_abc"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]string
	err := conn.do(t.Context(), validCreds(), http.MethodPost, "/v1/customers", map[string]string{"email": "test@example.com"}, &resp, "")
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
	if resp["id"] != "cus_abc" {
		t.Errorf("id = %q, want cus_abc", resp["id"])
	}
}

func TestDo_IdempotencyKey_SentOnPOST(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idemKey := r.Header.Get("Idempotency-Key")
		if idemKey == "" {
			t.Error("Idempotency-Key header missing on POST request")
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"key": idemKey})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]string
	err := conn.do(t.Context(), validCreds(), http.MethodPost, "/v1/refunds", map[string]string{"payment_intent": "pi_123"}, &resp, "test-idem-key")
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
	if resp["key"] != "test-idem-key" {
		t.Errorf("echoed key = %q, want test-idem-key", resp["key"])
	}
}

func TestDo_IdempotencyKey_NotSentOnGET(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if idemKey := r.Header.Get("Idempotency-Key"); idemKey != "" {
			t.Errorf("Idempotency-Key should not be sent on GET, got %q", idemKey)
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"id": "bal_123"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]string
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/v1/balance", nil, &resp, "should-be-ignored")
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
}

func TestDo_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "10")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"type":    "rate_limit_error",
				"message": "Too many requests",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/v1/balance", nil, nil, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) {
		if rle.RetryAfter != 10*time.Second {
			t.Errorf("RetryAfter = %v, want 10s", rle.RetryAfter)
		}
	}
}

func TestDo_AuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"type":    "authentication_error",
				"message": "Invalid API Key provided",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/v1/balance", nil, nil, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestDo_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(ctx, validCreds(), http.MethodGet, "/v1/balance", nil, nil, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}

func TestDo_MissingCredentials(t *testing.T) {
	t.Parallel()

	conn := New()
	err := conn.do(t.Context(), connectors.NewCredentials(map[string]string{}), http.MethodGet, "/v1/balance", nil, nil, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

// ---------------------------------------------------------------------------
// doGet / doPost convenience methods
// ---------------------------------------------------------------------------

func TestDoGet_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if idem := r.Header.Get("Idempotency-Key"); idem != "" {
			t.Errorf("doGet should not send Idempotency-Key, got %q", idem)
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"id": "bal_1"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]string
	err := conn.doGet(t.Context(), validCreds(), "/v1/balance", nil, &resp)
	if err != nil {
		t.Fatalf("doGet() unexpected error: %v", err)
	}
	if resp["id"] != "bal_1" {
		t.Errorf("id = %q, want bal_1", resp["id"])
	}
}

func TestDoPost_AutoIdempotency(t *testing.T) {
	t.Parallel()

	var capturedIdemKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedIdemKey = r.Header.Get("Idempotency-Key")

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"id": "cus_1"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	rawParams := json.RawMessage(`{"email":"test@example.com"}`)
	var resp map[string]string
	err := conn.doPost(t.Context(), validCreds(), "/v1/customers", map[string]string{"email": "test@example.com"}, &resp, "stripe.create_customer", rawParams)
	if err != nil {
		t.Fatalf("doPost() unexpected error: %v", err)
	}

	if capturedIdemKey == "" {
		t.Error("doPost should automatically derive and send Idempotency-Key")
	}

	// Verify determinism — same params produce same key.
	expectedKey := deriveIdempotencyKey("stripe.create_customer", rawParams)
	if capturedIdemKey != expectedKey {
		t.Errorf("Idempotency-Key = %q, want %q", capturedIdemKey, expectedKey)
	}
}

// ---------------------------------------------------------------------------
// Stripe-Version header
// ---------------------------------------------------------------------------

func TestDo_StripeVersionHeader(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ver := r.Header.Get("Stripe-Version")
		if ver == "" {
			t.Error("Stripe-Version header missing")
		}
		if ver != apiVersion {
			t.Errorf("Stripe-Version = %q, want %q", ver, apiVersion)
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/v1/balance", nil, nil, "")
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Error code inclusion in messages
// ---------------------------------------------------------------------------

func TestCheckResponse_IncludesErrorCode(t *testing.T) {
	t.Parallel()

	body := `{"error":{"type":"card_error","code":"card_declined","message":"Your card was declined"}}`
	err := checkResponse(402, http.Header{}, []byte(body))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "card_declined") {
		t.Errorf("error message should include code, got: %s", got)
	}
}

// ---------------------------------------------------------------------------
// Response body size limit
// ---------------------------------------------------------------------------

func TestDo_ResponseLimitReaderDoesNotBreakNormalResponses(t *testing.T) {
	t.Parallel()

	// Verify that the LimitReader doesn't interfere with normal-sized
	// responses (well under the 4 MB cap).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"id":"ok"}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]string
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/v1/balance", nil, &resp, "")
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
	if resp["id"] != "ok" {
		t.Errorf("id = %q, want ok", resp["id"])
	}
}

func TestDo_LargeResponseCappedByLimitReader(t *testing.T) {
	t.Parallel()

	// Return a response body larger than maxResponseBytes (4 MB).
	// The LimitReader should cap the read, and since the truncated body
	// won't be valid JSON, we expect an ExternalError from unmarshal.
	oversized := strings.Repeat("x", maxResponseBytes+1024)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(oversized))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]string
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/v1/test", nil, &resp, "")
	if err == nil {
		t.Fatal("expected error from oversized response unmarshal, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

// ---------------------------------------------------------------------------
// Error message truncation
// ---------------------------------------------------------------------------

func TestCheckResponse_LargeBodyTruncated(t *testing.T) {
	t.Parallel()

	// Simulate a non-JSON error response larger than maxErrorMessageBytes.
	largeBody := strings.Repeat("x", 1024)
	err := checkResponse(500, http.Header{}, []byte(largeBody))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// The error message should not contain the full 1024-byte body.
	if len(err.Error()) > 700 {
		t.Errorf("error message too large (%d bytes), should be truncated", len(err.Error()))
	}
}

// ---------------------------------------------------------------------------
// truncate (UTF-8 safety)
// ---------------------------------------------------------------------------

func TestTruncate_UTF8Safe(t *testing.T) {
	t.Parallel()

	// Each '日' is 3 bytes. With maxLen=5, naively slicing would cut the
	// second character in half, producing invalid UTF-8.
	result := truncate("日本語テスト", 5)
	// Should only include '日' (3 bytes) since '日本' (6 bytes) exceeds 5.
	if result != "日..." {
		t.Errorf("truncate = %q, want %q", result, "日...")
	}
}

// ---------------------------------------------------------------------------
// deriveIdempotencyKey
// ---------------------------------------------------------------------------

func TestDeriveIdempotencyKey_Deterministic(t *testing.T) {
	t.Parallel()

	params := json.RawMessage(`{"payment_intent":"pi_123","amount":500}`)
	key1 := deriveIdempotencyKey("stripe.issue_refund", params)
	key2 := deriveIdempotencyKey("stripe.issue_refund", params)

	if key1 != key2 {
		t.Errorf("same inputs produced different keys: %q vs %q", key1, key2)
	}
}

func TestDeriveIdempotencyKey_DifferentInputs(t *testing.T) {
	t.Parallel()

	key1 := deriveIdempotencyKey("stripe.issue_refund", json.RawMessage(`{"payment_intent":"pi_123"}`))
	key2 := deriveIdempotencyKey("stripe.issue_refund", json.RawMessage(`{"payment_intent":"pi_456"}`))
	key3 := deriveIdempotencyKey("stripe.create_customer", json.RawMessage(`{"payment_intent":"pi_123"}`))

	if key1 == key2 {
		t.Error("different params should produce different keys")
	}
	if key1 == key3 {
		t.Error("different action types should produce different keys")
	}
}

// ---------------------------------------------------------------------------
// Connector interface compliance
// ---------------------------------------------------------------------------

// Compile-time interface checks (matches GitHub/Slack connector pattern).
var _ connectors.Connector = (*StripeConnector)(nil)

func TestID(t *testing.T) {
	t.Parallel()

	conn := New()
	if conn.ID() != "stripe" {
		t.Errorf("ID() = %q, want %q", conn.ID(), "stripe")
	}
}

func TestActions_ReturnsMap(t *testing.T) {
	t.Parallel()

	conn := New()
	actions := conn.Actions()
	if actions == nil {
		t.Fatal("Actions() returned nil")
	}
	expected := []string{
		"stripe.create_customer",
		"stripe.create_invoice",
		"stripe.issue_refund",
		"stripe.list_subscriptions",
		"stripe.create_payment_link",
		"stripe.get_balance",
	}
	if len(actions) != len(expected) {
		t.Errorf("Actions() returned %d actions, want %d", len(actions), len(expected))
	}
	for _, name := range expected {
		if _, ok := actions[name]; !ok {
			t.Errorf("Actions() missing %q", name)
		}
	}
}
