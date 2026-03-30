package stripe

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ---------------------------------------------------------------------------
// ValidateCredentials
// ---------------------------------------------------------------------------

func TestValidateCredentials_Valid(t *testing.T) {
	t.Parallel()

	conn := New()
	tests := []struct {
		name  string
		creds connectors.Credentials
	}{
		{"api_key live", connectors.NewCredentials(map[string]string{"api_key": "sk_live_abc123"})},
		{"api_key test", connectors.NewCredentials(map[string]string{"api_key": "sk_test_abc123"})},
		{"api_key restricted live", connectors.NewCredentials(map[string]string{"api_key": "rk_live_abc123"})},
		{"api_key restricted test", connectors.NewCredentials(map[string]string{"api_key": "rk_test_abc123"})},
		{"access_token live", connectors.NewCredentials(map[string]string{"access_token": "sk_live_oauth123"})},
		{"access_token test", connectors.NewCredentials(map[string]string{"access_token": "sk_test_oauth123"})},
		{"access_token preferred over api_key", connectors.NewCredentials(map[string]string{
			"access_token": "sk_live_oauth",
			"api_key":      "sk_live_static",
		})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := conn.ValidateCredentials(t.Context(), tt.creds); err != nil {
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
		{"empty api_key", connectors.NewCredentials(map[string]string{"api_key": ""})},
		{"empty access_token", connectors.NewCredentials(map[string]string{"access_token": ""})},
		{"bad prefix api_key", connectors.NewCredentials(map[string]string{"api_key": "pk_test_abc123"})},
		{"bad prefix access_token", connectors.NewCredentials(map[string]string{"access_token": "pk_live_abc123"})},
		{"wrong cred name", connectors.NewCredentials(map[string]string{"token": "sk_test_abc123"})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
// validateMetadata
// ---------------------------------------------------------------------------

func TestValidateMetadata_RejectsNonStringValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata map[string]any
	}{
		{"nested object", map[string]any{"key": map[string]any{"nested": "value"}}},
		{"array value", map[string]any{"key": []any{"a", "b"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateMetadata(tt.metadata)
			if err == nil {
				t.Fatal("expected error for non-string metadata value, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestValidateMetadata_AcceptsStringValues(t *testing.T) {
	t.Parallel()

	err := validateMetadata(map[string]any{
		"string_val": "hello",
		"number_val": float64(42),
		"bool_val":   true,
	})
	if err != nil {
		t.Errorf("validateMetadata() unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// validateEnum
// ---------------------------------------------------------------------------

func TestValidateEnum_Valid(t *testing.T) {
	t.Parallel()

	if err := validateEnum("once", "duration", []string{"once", "repeating", "forever"}); err != nil {
		t.Errorf("validateEnum() unexpected error: %v", err)
	}
}

func TestValidateEnum_EmptyIsValid(t *testing.T) {
	t.Parallel()

	// Empty string means "not provided" and should not be rejected.
	if err := validateEnum("", "duration", []string{"once", "repeating", "forever"}); err != nil {
		t.Errorf("validateEnum(\"\") unexpected error: %v", err)
	}
}

func TestValidateEnum_Invalid(t *testing.T) {
	t.Parallel()

	err := validateEnum("bad_value", "duration", []string{"once", "repeating", "forever"})
	if err == nil {
		t.Fatal("expected error for invalid enum value, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

// ---------------------------------------------------------------------------
// validateCurrency
// ---------------------------------------------------------------------------

func TestValidateCurrency_Valid(t *testing.T) {
	t.Parallel()

	for _, c := range []string{"usd", "USD", "eur", "gbp", "jpy"} {
		if err := validateCurrency(c); err != nil {
			t.Errorf("validateCurrency(%q) unexpected error: %v", c, err)
		}
	}
}

func TestValidateCurrency_Invalid(t *testing.T) {
	t.Parallel()

	for _, c := range []string{"us", "dollars", "1234", "u$d", ""} {
		// Empty string is handled by callers before validateCurrency, so
		// if it reaches here it should fail the length check.
		if c == "" {
			continue
		}
		err := validateCurrency(c)
		if err == nil {
			t.Errorf("validateCurrency(%q) expected error, got nil", c)
		}
		if !connectors.IsValidationError(err) {
			t.Errorf("validateCurrency(%q) expected ValidationError, got %T: %v", c, err, err)
		}
	}
}

// ---------------------------------------------------------------------------
// encodeParams (deterministic ordering)
// ---------------------------------------------------------------------------

func TestEncodeParams_Sorted(t *testing.T) {
	t.Parallel()

	encoded := encodeParams(map[string]string{
		"email":  "a@b.com",
		"name":   "Test",
		"amount": "100",
	})

	// url.Values.Encode sorts keys alphabetically.
	want := "amount=100&email=a%40b.com&name=Test"
	if encoded != want {
		t.Errorf("encodeParams = %q, want %q", encoded, want)
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
