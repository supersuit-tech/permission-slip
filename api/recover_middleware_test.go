package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPanicRecoverMiddleware_ReturnsStructured500(t *testing.T) {
	// Do not use t.Parallel: these tests swap the package-level panicCaptureError
	// and would race with each other.

	inner := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("super-secret-panic-detail-do-not-leak")
	})

	var sentryCalled bool
	oldCapture := panicCaptureError
	panicCaptureError = func(_ context.Context, _ error) {
		sentryCalled = true
	}
	t.Cleanup(func() { panicCaptureError = oldCapture })

	handler := TraceIDMiddleware(PanicRecoverMiddleware(inner))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if !sentryCalled {
		t.Error("expected CaptureError to be invoked")
	}

	var parsed ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&parsed); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if parsed.Error.Code != ErrInternalPanic {
		t.Errorf("error.code = %q, want %q", parsed.Error.Code, ErrInternalPanic)
	}
	if parsed.Error.Retryable {
		t.Error("expected retryable false")
	}
	if parsed.Error.TraceID == "" {
		t.Error("expected non-empty trace_id on error")
	}
	body := rec.Body.String()
	if strings.Contains(body, "super-secret-panic-detail-do-not-leak") {
		t.Error("panic value leaked into response body")
	}
}

func TestPanicRecoverMiddleware_NoSecondWriteAfterImplicit200(t *testing.T) {
	// See TestPanicRecoverMiddleware_ReturnsStructured500 — package-level hook.

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
		panic("after implicit headers")
	})

	var sentryCalled bool
	oldCapture := panicCaptureError
	panicCaptureError = func(_ context.Context, _ error) {
		sentryCalled = true
	}
	t.Cleanup(func() { panicCaptureError = oldCapture })

	handler := TraceIDMiddleware(PanicRecoverMiddleware(inner))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !sentryCalled {
		t.Error("expected CaptureError to be invoked")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (implicit from Write)", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != "ok" {
		t.Errorf("body = %q, want ok", rec.Body.String())
	}
}

func TestPanicRecoverMiddleware_NoSecondWriteAfterHeaders(t *testing.T) {
	// See TestPanicRecoverMiddleware_ReturnsStructured500 — package-level hook.

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		panic("after headers")
	})

	var sentryCalled bool
	oldCapture := panicCaptureError
	panicCaptureError = func(_ context.Context, _ error) {
		sentryCalled = true
	}
	t.Cleanup(func() { panicCaptureError = oldCapture })

	handler := TraceIDMiddleware(PanicRecoverMiddleware(inner))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !sentryCalled {
		t.Error("expected CaptureError to be invoked")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (no overwrite after headers)", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != "" {
		t.Errorf("expected empty body after panic with headers sent, got %q", rec.Body.String())
	}
}

func TestPanicRecoverMiddleware_PassThrough(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	handler := TraceIDMiddleware(PanicRecoverMiddleware(inner))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
}
