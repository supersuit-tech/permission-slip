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

func TestDo_ContextCanceled(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := conn.do(ctx, validCreds(), http.MethodGet, "/test", nil)
	if err == nil {
		t.Fatal("do() expected error with canceled context, got nil")
	}
	if !connectors.IsCanceledError(err) {
		t.Errorf("expected CanceledError, got %T: %v", err, err)
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

// TestManifest_ActionsMatchRegistered ensures the manifest action schemas
// stay in sync with the registered Action handlers. If you add a new action
// to Actions() you must also add a schema in Manifest(), and vice versa.
func TestManifest_ActionsMatchRegistered(t *testing.T) {
	t.Parallel()
	c := New()
	manifest := c.Manifest()
	actions := c.Actions()

	for _, a := range manifest.Actions {
		if _, ok := actions[a.ActionType]; !ok {
			t.Errorf("manifest action %q not registered in Actions()", a.ActionType)
		}
	}
	if len(actions) != len(manifest.Actions) {
		t.Errorf("Actions() has %d entries but Manifest() has %d", len(actions), len(manifest.Actions))
	}
}
