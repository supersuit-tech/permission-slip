package quickbooks

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
	creds := connectors.NewCredentials(map[string]string{
		"access_token": "test-token",
		"realm_id":     "123456",
	})
	if err := conn.ValidateCredentials(t.Context(), creds); err != nil {
		t.Errorf("ValidateCredentials() unexpected error: %v", err)
	}
}

func TestValidateCredentials_Invalid(t *testing.T) {
	t.Parallel()

	conn := New()
	tests := []struct {
		name  string
		creds connectors.Credentials
	}{
		{"missing access_token", connectors.NewCredentials(map[string]string{"realm_id": "123"})},
		{"empty access_token", connectors.NewCredentials(map[string]string{"access_token": "", "realm_id": "123"})},
		{"missing realm_id", connectors.NewCredentials(map[string]string{"access_token": "tok"})},
		{"empty realm_id", connectors.NewCredentials(map[string]string{"access_token": "tok", "realm_id": ""})},
		{"no creds", connectors.NewCredentials(map[string]string{})},
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
// doJSON integration tests via httptest
// ---------------------------------------------------------------------------

func TestDoJSON_GET_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-abc123" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-access-token-abc123")
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q, want application/json", got)
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{"id": "123"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.doGet(t.Context(), validCreds(), "/v3/company/1234567890/account", &resp)
	if err != nil {
		t.Fatalf("doGet() unexpected error: %v", err)
	}
	if resp["id"] != "123" {
		t.Errorf("id = %v, want 123", resp["id"])
	}
}

func TestDoJSON_POST_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["DisplayName"] != "Acme Corp" {
			t.Errorf("DisplayName = %v, want Acme Corp", body["DisplayName"])
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{"Customer": map[string]any{"Id": "42"}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.doPost(t.Context(), validCreds(), "/v3/company/1234567890/customer", map[string]any{"DisplayName": "Acme Corp"}, &resp)
	if err != nil {
		t.Fatalf("doPost() unexpected error: %v", err)
	}
}

func TestDoJSON_AuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"Fault": map[string]any{
				"Error": []map[string]any{
					{"Message": "AuthenticationFailed", "code": "100"},
				},
				"type": "AUTHENTICATION",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.doGet(t.Context(), validCreds(), "/v3/company/123/test", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestDoJSON_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"message":"rate limit exceeded"}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.doGet(t.Context(), validCreds(), "/v3/company/123/test", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) {
		if rle.RetryAfter != 30*time.Second {
			t.Errorf("RetryAfter = %v, want 30s", rle.RetryAfter)
		}
	}
}

func TestDoJSON_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.doGet(ctx, validCreds(), "/v3/company/123/test", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}

func TestDoJSON_MissingCredentials(t *testing.T) {
	t.Parallel()

	conn := New()
	err := conn.doJSON(t.Context(), connectors.NewCredentials(map[string]string{}), http.MethodGet, "/test", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestDoJSON_ValidationError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"Fault": map[string]any{
				"Error": []map[string]any{
					{"Message": "Invalid request", "Detail": "CustomerRef is required", "code": "2050"},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.doGet(t.Context(), validCreds(), "/v3/company/123/test", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

// ---------------------------------------------------------------------------
// checkResponse
// ---------------------------------------------------------------------------

func TestCheckResponse_Success(t *testing.T) {
	t.Parallel()

	if err := checkResponse(200, http.Header{}, []byte(`{}`)); err != nil {
		t.Errorf("checkResponse(200) unexpected error: %v", err)
	}
}

func TestCheckResponse_403Forbidden(t *testing.T) {
	t.Parallel()

	err := checkResponse(403, http.Header{}, []byte(`Forbidden`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError for 403, got %T: %v", err, err)
	}
}

func TestCheckResponse_500ServerError(t *testing.T) {
	t.Parallel()

	err := checkResponse(500, http.Header{}, []byte(`Internal Server Error`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestCheckResponse_FaultErrorIncludesCode(t *testing.T) {
	t.Parallel()

	body := `{"Fault":{"Error":[{"Message":"Object Not Found","Detail":"Something was not found","code":"610"}],"type":"ValidationFault"}}`
	err := checkResponse(400, http.Header{}, []byte(body))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "610") {
		t.Errorf("error should include code 610, got: %s", got)
	}
}

func TestCheckResponse_LargeBodyTruncated(t *testing.T) {
	t.Parallel()

	largeBody := strings.Repeat("x", 1024)
	err := checkResponse(500, http.Header{}, []byte(largeBody))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(err.Error()) > 700 {
		t.Errorf("error message too large (%d bytes), should be truncated", len(err.Error()))
	}
}

// ---------------------------------------------------------------------------
// Connector interface compliance
// ---------------------------------------------------------------------------

var _ connectors.Connector = (*QuickBooksConnector)(nil)
var _ connectors.ManifestProvider = (*QuickBooksConnector)(nil)

func TestManifest_Valid(t *testing.T) {
	t.Parallel()

	conn := New()
	m := conn.Manifest()
	if err := m.Validate(); err != nil {
		t.Fatalf("Manifest().Validate() error: %v", err)
	}
	if m.ID != "quickbooks" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "quickbooks")
	}
	if len(m.Actions) != 8 {
		t.Errorf("Manifest().Actions has %d entries, want 8", len(m.Actions))
	}
	if len(m.RequiredCredentials) != 1 {
		t.Errorf("Manifest().RequiredCredentials has %d entries, want 1", len(m.RequiredCredentials))
	}
	if m.RequiredCredentials[0].AuthType != "oauth2" {
		t.Errorf("RequiredCredentials[0].AuthType = %q, want %q", m.RequiredCredentials[0].AuthType, "oauth2")
	}
	if len(m.Templates) != 8 {
		t.Errorf("Manifest().Templates has %d entries, want 8", len(m.Templates))
	}
	if len(m.OAuthProviders) != 1 {
		t.Errorf("Manifest().OAuthProviders has %d entries, want 1", len(m.OAuthProviders))
	}
}

func TestManifest_ActionsMatchRegistered(t *testing.T) {
	t.Parallel()

	conn := New()
	m := conn.Manifest()
	actions := conn.Actions()

	for _, ma := range m.Actions {
		if _, ok := actions[ma.ActionType]; !ok {
			t.Errorf("Manifest action %q not found in Actions() map", ma.ActionType)
		}
	}
	for actionType := range actions {
		found := false
		for _, ma := range m.Actions {
			if ma.ActionType == actionType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Actions() has %q but it's not in the Manifest", actionType)
		}
	}
}

func TestID(t *testing.T) {
	t.Parallel()

	conn := New()
	if conn.ID() != "quickbooks" {
		t.Errorf("ID() = %q, want %q", conn.ID(), "quickbooks")
	}
}

func TestActions_ReturnsMap(t *testing.T) {
	t.Parallel()

	conn := New()
	actions := conn.Actions()
	if actions == nil {
		t.Fatal("Actions() returned nil")
	}
	expectedActions := []string{
		"quickbooks.create_invoice",
		"quickbooks.record_payment",
		"quickbooks.create_expense",
		"quickbooks.get_profit_loss",
		"quickbooks.get_balance_sheet",
		"quickbooks.reconcile_transaction",
		"quickbooks.create_customer",
		"quickbooks.list_accounts",
	}
	if len(actions) != len(expectedActions) {
		t.Errorf("Actions() returned %d actions, want %d", len(actions), len(expectedActions))
	}
	for _, name := range expectedActions {
		if _, ok := actions[name]; !ok {
			t.Errorf("Actions() missing %q", name)
		}
	}
}
