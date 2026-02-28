package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// echoHandler returns 200 OK with a fixed body so tests can confirm the
// request reached the inner handler.
var echoHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
})

func TestCORS_NoOriginPassesThrough(t *testing.T) {
	t.Parallel()
	handler := CORSMiddleware([]string{"https://app.example.com"})(echoHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("CORS headers should not be set for requests without Origin")
	}
}

func TestCORS_AllowedOriginSetsHeaders(t *testing.T) {
	t.Parallel()
	allowed := "https://app.example.com"
	handler := CORSMiddleware([]string{allowed})(echoHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.Header.Set("Origin", allowed)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != allowed {
		t.Fatalf("expected Access-Control-Allow-Origin=%q, got %q", allowed, got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected Access-Control-Allow-Credentials=true, got %q", got)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("expected Vary to contain Origin, got %q", got)
	}
}

func TestCORS_DisallowedOriginBlocked(t *testing.T) {
	t.Parallel()
	handler := CORSMiddleware([]string{"https://app.example.com"})(echoHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestCORS_PreflightReturnsNoContent(t *testing.T) {
	t.Parallel()
	allowed := "https://app.example.com"
	handler := CORSMiddleware([]string{allowed})(echoHandler)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/agents", nil)
	req.Header.Set("Origin", allowed)
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for preflight, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("expected Access-Control-Allow-Methods to be set on preflight")
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatal("expected Access-Control-Allow-Headers to be set on preflight")
	}
	if got := rec.Header().Get("Access-Control-Max-Age"); got != "86400" {
		t.Fatalf("expected Access-Control-Max-Age=86400, got %q", got)
	}
	if body := rec.Body.String(); body != "" {
		t.Fatalf("preflight should have empty body, got %q", body)
	}
}

func TestCORS_OptionsWithoutACRM_PassesThrough(t *testing.T) {
	t.Parallel()
	allowed := "https://app.example.com"
	handler := CORSMiddleware([]string{allowed})(echoHandler)

	// OPTIONS request with Origin but no Access-Control-Request-Method is
	// NOT a CORS preflight — it should reach the inner handler.
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/agents", nil)
	req.Header.Set("Origin", allowed)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for non-preflight OPTIONS, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "ok" {
		t.Fatalf("expected inner handler body, got %q", body)
	}
}

func TestCORS_EmptyAllowList_SameOriginAllowed(t *testing.T) {
	t.Parallel()
	handler := CORSMiddleware(nil)(echoHandler)

	// httptest.NewRequest defaults Host to "example.com" with no TLS → http://example.com
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profile", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for same-origin request with empty allow-list, got %d", rec.Code)
	}
}

func TestCORS_EmptyAllowList_CrossOriginBlocked(t *testing.T) {
	t.Parallel()
	handler := CORSMiddleware(nil)(echoHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when no origins configured and origin differs, got %d", rec.Code)
	}
}

func TestCORS_MultipleAllowedOrigins(t *testing.T) {
	t.Parallel()
	origins := []string{
		"https://app.example.com",
		"https://staging.example.com",
	}
	handler := CORSMiddleware(origins)(echoHandler)

	for _, origin := range origins {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
		req.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("origin %q: expected 200, got %d", origin, rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Fatalf("origin %q: expected ACAO=%q, got %q", origin, origin, got)
		}
	}
}
