package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGatewaySecretMiddleware_NoSecret_PassesThrough(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	handler := GatewaySecretMiddleware("")(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 when no secret configured, got %d", rec.Code)
	}
}

func TestGatewaySecretMiddleware_ValidSecret_PassesThrough(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	handler := GatewaySecretMiddleware("my-secret-key")(inner)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("X-Gateway-Secret", "my-secret-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 with valid secret, got %d", rec.Code)
	}
}

func TestGatewaySecretMiddleware_InvalidSecret_Rejects(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := GatewaySecretMiddleware("my-secret-key")(inner)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("X-Gateway-Secret", "wrong-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 with invalid secret, got %d", rec.Code)
	}
}

func TestGatewaySecretMiddleware_MissingHeader_Rejects(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := GatewaySecretMiddleware("my-secret-key")(inner)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 with missing header, got %d", rec.Code)
	}
}

func TestGatewaySecretMiddleware_OptionsExempt(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	handler := GatewaySecretMiddleware("my-secret-key")(inner)
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/test", nil)
	// No X-Gateway-Secret header — should still pass through.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 for OPTIONS (preflight exempt), got %d", rec.Code)
	}
}

func TestGatewaySecretMiddleware_EmptyProvided_Rejects(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := GatewaySecretMiddleware("my-secret-key")(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Gateway-Secret", "")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 with empty header value, got %d", rec.Code)
	}
}

func TestGatewaySecretMiddleware_TimingSafe(t *testing.T) {
	t.Parallel()
	// This test verifies that both a completely wrong key and a partially
	// correct key are both rejected — not that timing is identical, but that
	// the comparison uses constant-time logic (verified by code inspection).
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := GatewaySecretMiddleware("abcdefghijklmnop")(inner)

	for _, provided := range []string{"abcdefghijklmnox", "xxxxxxxxxxxxxxxx", "abc", ""} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Gateway-Secret", provided)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 for %q, got %d", provided, rec.Code)
		}
	}
}

func TestGatewaySecretMiddleware_AllHTTPMethods(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := GatewaySecretMiddleware("test-secret")(inner)

	// All non-OPTIONS methods should require the secret.
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead} {
		req := httptest.NewRequest(method, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("%s without secret: expected 403, got %d", method, rec.Code)
		}

		req = httptest.NewRequest(method, "/", nil)
		req.Header.Set("X-Gateway-Secret", "test-secret")
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("%s with valid secret: expected 200, got %d", method, rec.Code)
		}
	}
}
