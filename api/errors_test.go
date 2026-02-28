package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestErrorResponseJSON(t *testing.T) {
	t.Parallel()
	e := ErrorResponse{
		Error: Error{
			Code:      ErrInvalidSignature,
			Message:   "Signature verification failed",
			Retryable: false,
			Details: map[string]any{
				"timestamp":       "1770816000",
				"expected_window": "300 seconds",
			},
			TraceID: "trace_xyz789",
		},
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("failed to marshal ErrorResponse: %v", err)
	}

	var parsed ErrorResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if parsed.Error.Code != ErrInvalidSignature {
		t.Errorf("expected code %q, got %q", ErrInvalidSignature, parsed.Error.Code)
	}
	if parsed.Error.Message != "Signature verification failed" {
		t.Errorf("expected message 'Signature verification failed', got %q", parsed.Error.Message)
	}
	if parsed.Error.Retryable != false {
		t.Errorf("expected retryable false, got %v", parsed.Error.Retryable)
	}
	if parsed.Error.TraceID != "trace_xyz789" {
		t.Errorf("expected trace_id 'trace_xyz789', got %q", parsed.Error.TraceID)
	}
	if parsed.Error.Details["timestamp"] != "1770816000" {
		t.Errorf("expected detail timestamp '1770816000', got %v", parsed.Error.Details["timestamp"])
	}
	if parsed.Error.Details["expected_window"] != "300 seconds" {
		t.Errorf("expected detail expected_window '300 seconds', got %v", parsed.Error.Details["expected_window"])
	}
}

func TestErrorResponseOmitsEmptyOptionalFields(t *testing.T) {
	t.Parallel()
	e := ErrorResponse{
		Error: Error{
			Code:      ErrInvalidRequest,
			Message:   "Bad request",
			Retryable: false,
		},
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("failed to marshal ErrorResponse: %v", err)
	}

	// Use raw map to verify JSON key presence/absence — the typed struct
	// would silently zero-fill missing fields, hiding omitempty bugs.
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	errorObj := raw["error"].(map[string]any)

	if _, exists := errorObj["details"]; exists {
		t.Error("expected 'details' to be omitted when nil")
	}
	if _, exists := errorObj["trace_id"]; exists {
		t.Error("expected 'trace_id' to be omitted when empty")
	}
	if _, exists := errorObj["retry_after"]; exists {
		t.Error("expected 'retry_after' to be omitted when zero")
	}
}

func TestErrorResponseIncludesRetryAfter(t *testing.T) {
	t.Parallel()
	e := TooManyRequests("Rate limit exceeded", 60)

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("failed to marshal ErrorResponse: %v", err)
	}

	var parsed ErrorResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if parsed.Error.RetryAfter != 60 {
		t.Errorf("expected retry_after 60, got %d", parsed.Error.RetryAfter)
	}
	if parsed.Error.Retryable != true {
		t.Errorf("expected retryable true, got %v", parsed.Error.Retryable)
	}
}

func TestRespondError(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	e := BadRequest(ErrInvalidRequest, "Missing required field")

	RespondError(w, r, http.StatusBadRequest, e)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", ct)
	}

	var parsed ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	if parsed.Error.Code != ErrInvalidRequest {
		t.Errorf("expected code %q, got %q", ErrInvalidRequest, parsed.Error.Code)
	}
	if parsed.Error.Message != "Missing required field" {
		t.Errorf("expected message 'Missing required field', got %q", parsed.Error.Message)
	}
	if parsed.Error.Retryable != false {
		t.Errorf("expected retryable false, got %v", parsed.Error.Retryable)
	}

	// Without TraceIDMiddleware, trace_id must be absent from JSON.
	if strings.Contains(w.Body.String(), "trace_id") {
		t.Error("expected trace_id to be absent when no trace ID in context")
	}
}

func TestRespondError_InjectsTraceID(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	// Simulate TraceIDMiddleware by adding trace ID to context.
	ctx := context.WithValue(r.Context(), traceIDKey{}, "trace_test123")
	r = r.WithContext(ctx)

	e := InternalError("something went wrong")
	RespondError(w, r, http.StatusInternalServerError, e)

	var parsed ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	if parsed.Error.TraceID != "trace_test123" {
		t.Errorf("expected trace_id 'trace_test123', got %q", parsed.Error.TraceID)
	}
	// Verify the rest of the response wasn't corrupted by injection.
	if parsed.Error.Code != ErrInternalError {
		t.Errorf("expected code %q, got %q", ErrInternalError, parsed.Error.Code)
	}
	if parsed.Error.Message != "something went wrong" {
		t.Errorf("expected message 'something went wrong', got %q", parsed.Error.Message)
	}
}

func TestRespondError_DoesNotOverrideExplicitTraceID(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	ctx := context.WithValue(r.Context(), traceIDKey{}, "trace_from_middleware")
	r = r.WithContext(ctx)

	e := ErrorResponse{
		Error: Error{
			Code:      ErrInternalError,
			Message:   "oops",
			Retryable: true,
			TraceID:   "trace_explicit",
		},
	}
	RespondError(w, r, http.StatusInternalServerError, e)

	var parsed ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	if parsed.Error.TraceID != "trace_explicit" {
		t.Errorf("expected trace_id 'trace_explicit', got %q", parsed.Error.TraceID)
	}
}

func TestTraceIDMiddleware(t *testing.T) {
	t.Parallel()
	var capturedTraceID string
	handler := TraceIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTraceID = TraceID(r.Context())
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)

	if capturedTraceID == "" {
		t.Fatal("expected trace ID to be set in context")
	}
	if !strings.HasPrefix(capturedTraceID, "trace_") {
		t.Errorf("expected trace ID to start with 'trace_', got %q", capturedTraceID)
	}
	// "trace_" (6 chars) + 32 hex chars = 38
	if len(capturedTraceID) != 38 {
		t.Errorf("expected trace ID length 38, got %d (%q)", len(capturedTraceID), capturedTraceID)
	}
}

func TestTraceIDMiddleware_UniquenessAcrossRequests(t *testing.T) {
	t.Parallel()
	ids := make(map[string]struct{}, 100)
	handler := TraceIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ids[TraceID(r.Context())] = struct{}{}
	}))

	for range 100 {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(w, r)
	}

	if len(ids) != 100 {
		t.Errorf("expected 100 unique trace IDs, got %d", len(ids))
	}
}

func TestTraceID_BareContext(t *testing.T) {
	t.Parallel()
	id := TraceID(context.Background())
	if id != "" {
		t.Errorf("expected empty string for bare context, got %q", id)
	}
}

func TestConvenienceConstructors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		resp       ErrorResponse
		wantCode   ErrorCode
		wantMsg    string
		retryable  bool
		retryAfter int
	}{
		{"BadRequest", BadRequest(ErrInvalidRequest, "bad"), ErrInvalidRequest, "bad", false, 0},
		{"BadRequest_CustomCode", BadRequest(ErrInvalidPublicKey, "bad key"), ErrInvalidPublicKey, "bad key", false, 0},
		{"Unauthorized", Unauthorized(ErrInvalidSignature, "unauth"), ErrInvalidSignature, "unauth", false, 0},
		{"Forbidden", Forbidden(ErrAgentNotAuthorized, "denied"), ErrAgentNotAuthorized, "denied", false, 0},
		{"NotFound", NotFound(ErrAgentNotFound, "missing"), ErrAgentNotFound, "missing", false, 0},
		{"NotFound_CustomCode", NotFound(ErrApprovalNotFound, "no approval"), ErrApprovalNotFound, "no approval", false, 0},
		{"Conflict", Conflict(ErrDuplicateRequestID, "dup"), ErrDuplicateRequestID, "dup", false, 0},
		{"Conflict_CustomCode", Conflict(ErrAgentAlreadyRegistered, "exists"), ErrAgentAlreadyRegistered, "exists", false, 0},
		{"Gone", Gone(ErrApprovalExpired, "expired"), ErrApprovalExpired, "expired", false, 0},
		{"Gone_CustomCode", Gone(ErrRegistrationExpired, "reg expired"), ErrRegistrationExpired, "reg expired", false, 0},
		{"TooManyRequests", TooManyRequests("slow down", 30), ErrRateLimited, "slow down", true, 30},
		{"InternalError", InternalError("oops"), ErrInternalError, "oops", true, 0},
		{"ServiceUnavailable", ServiceUnavailable("down"), ErrServiceUnavailable, "down", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.resp.Error.Code != tt.wantCode {
				t.Errorf("code: want %q, got %q", tt.wantCode, tt.resp.Error.Code)
			}
			if tt.resp.Error.Message != tt.wantMsg {
				t.Errorf("message: want %q, got %q", tt.wantMsg, tt.resp.Error.Message)
			}
			if tt.resp.Error.Retryable != tt.retryable {
				t.Errorf("retryable: want %v, got %v", tt.retryable, tt.resp.Error.Retryable)
			}
			if tt.resp.Error.RetryAfter != tt.retryAfter {
				t.Errorf("retry_after: want %d, got %d", tt.retryAfter, tt.resp.Error.RetryAfter)
			}
		})
	}
}
