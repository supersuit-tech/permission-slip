package walmart

import (
	"context"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "walmart" {
		t.Errorf("ID() = %q, want %q", got, "walmart")
	}
}

func TestValidateCredentials_Valid(t *testing.T) {
	t.Parallel()
	c := New()
	err := c.ValidateCredentials(context.Background(), validCreds())
	if err != nil {
		t.Fatalf("ValidateCredentials() unexpected error: %v", err)
	}
}

func TestValidateCredentials_MissingConsumerID(t *testing.T) {
	t.Parallel()
	c := New()
	err := c.ValidateCredentials(context.Background(), connectors.NewCredentials(map[string]string{
		"private_key": testPrivateKeyPEM(),
	}))
	if err == nil {
		t.Fatal("ValidateCredentials() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestValidateCredentials_MissingPrivateKey(t *testing.T) {
	t.Parallel()
	c := New()
	err := c.ValidateCredentials(context.Background(), connectors.NewCredentials(map[string]string{
		"consumer_id": "test-consumer-id",
	}))
	if err == nil {
		t.Fatal("ValidateCredentials() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestValidateCredentials_InvalidPrivateKey(t *testing.T) {
	t.Parallel()
	c := New()
	err := c.ValidateCredentials(context.Background(), connectors.NewCredentials(map[string]string{
		"consumer_id": "test-consumer-id",
		"private_key": "not-a-pem-key",
	}))
	if err == nil {
		t.Fatal("ValidateCredentials() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestDo_SetsHeaders(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("WM_CONSUMER.ID"); got != "test-consumer-id" {
			t.Errorf("WM_CONSUMER.ID = %q, want %q", got, "test-consumer-id")
		}
		if got := r.Header.Get("WM_SEC.KEY_VERSION"); got != "1" {
			t.Errorf("WM_SEC.KEY_VERSION = %q, want %q", got, "1")
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q, want %q", got, "application/json")
		}
		if got := r.Header.Get("WM_CONSUMER.INTIMESTAMP"); got == "" {
			t.Error("WM_CONSUMER.INTIMESTAMP header missing")
		}
		if got := r.Header.Get("WM_SEC.AUTH_SIGNATURE"); got == "" {
			t.Error("WM_SEC.AUTH_SIGNATURE header missing")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]interface{}
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/test", &resp)
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
}

func TestSignRequest(t *testing.T) {
	t.Parallel()

	sig, err := signRequest("test-consumer-id", "1234567890000", "1", testPrivateKeyPEM())
	if err != nil {
		t.Fatalf("signRequest() unexpected error: %v", err)
	}
	if sig == "" {
		t.Error("signRequest() returned empty signature")
	}
}

func TestSignRequest_InvalidKey(t *testing.T) {
	t.Parallel()

	_, err := signRequest("test-consumer-id", "1234567890000", "1", "not-a-pem-key")
	if err == nil {
		t.Fatal("signRequest() expected error, got nil")
	}
}

func TestDo_MissingConsumerID(t *testing.T) {
	t.Parallel()

	conn := New()
	err := conn.do(t.Context(), connectors.NewCredentials(map[string]string{}), http.MethodGet, "/test", nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestDo_DefaultKeyVersion(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("WM_SEC.KEY_VERSION"); got != "1" {
			t.Errorf("WM_SEC.KEY_VERSION = %q, want %q (default)", got, "1")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	creds := connectors.NewCredentials(map[string]string{
		"consumer_id": "test-consumer-id",
		"private_key": testPrivateKeyPEM(),
	})
	var resp map[string]interface{}
	err := conn.do(t.Context(), creds, http.MethodGet, "/test", &resp)
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
}

func TestManifest_Valid(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()
	if err := m.Validate(); err != nil {
		t.Fatalf("Manifest().Validate() error: %v", err)
	}
}

func TestActions_AllRegistered(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()
	expected := []string{
		"walmart.search_products",
		"walmart.get_product",
		"walmart.get_taxonomy",
		"walmart.get_trending",
	}
	for _, actionType := range expected {
		if _, ok := actions[actionType]; !ok {
			t.Errorf("missing action %q", actionType)
		}
	}
	if len(actions) != len(expected) {
		t.Errorf("got %d actions, want %d", len(actions), len(expected))
	}
}
