package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestLoggerMiddleware_LogsFields(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with TraceID first (sets trace_id in context), then RequestLogger.
	handler := TraceIDMiddleware(RequestLoggerMiddleware(logger, "")(inner))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log entry: %v (raw: %s)", err, buf.String())
	}

	if entry["method"] != "GET" {
		t.Errorf("expected method GET, got %v", entry["method"])
	}
	// path uses RequestURI to preserve the full path even behind StripPrefix.
	if entry["path"] != "/api/v1/agents" {
		t.Errorf("expected path /api/v1/agents, got %v", entry["path"])
	}
	// status is logged as a float64 by encoding/json
	if status, ok := entry["status"].(float64); !ok || int(status) != 200 {
		t.Errorf("expected status 200, got %v", entry["status"])
	}
	if _, ok := entry["duration"]; !ok {
		t.Error("expected duration field in log entry")
	}
	if entry["trace_id"] == nil || entry["trace_id"] == "" {
		t.Error("expected non-empty trace_id in log entry")
	}
	if entry["msg"] != "http request" {
		t.Errorf("expected msg 'http request', got %v", entry["msg"])
	}
}

func TestRequestLoggerMiddleware_Captures4xxStatus(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	handler := RequestLoggerMiddleware(logger, "")(inner)
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log entry: %v", err)
	}

	if status, ok := entry["status"].(float64); !ok || int(status) != 404 {
		t.Errorf("expected status 404, got %v", entry["status"])
	}
	if entry["level"] != "WARN" {
		t.Errorf("expected log level WARN for 4xx, got %v", entry["level"])
	}
}

func TestRequestLoggerMiddleware_Captures5xxStatus(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	handler := RequestLoggerMiddleware(logger, "")(inner)
	req := httptest.NewRequest(http.MethodPost, "/error", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log entry: %v", err)
	}

	if status, ok := entry["status"].(float64); !ok || int(status) != 500 {
		t.Errorf("expected status 500, got %v", entry["status"])
	}
	if entry["level"] != "ERROR" {
		t.Errorf("expected log level ERROR for 5xx, got %v", entry["level"])
	}
}

func TestRequestLoggerMiddleware_IncludesClientIP(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RequestLoggerMiddleware(logger, "")(inner)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.RemoteAddr = "10.0.0.42:54321"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log entry: %v", err)
	}

	if entry["client_ip"] != "10.0.0.42" {
		t.Errorf("expected client_ip '10.0.0.42', got %v", entry["client_ip"])
	}
}

func TestStatusRecorder_DefaultsTo200(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	sr := &statusRecorder{ResponseWriter: rec, status: http.StatusOK}

	// If WriteHeader is never called, status should remain 200.
	if sr.status != 200 {
		t.Errorf("expected default status 200, got %d", sr.status)
	}
}

func TestStatusRecorder_CapturesWriteHeader(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	sr := &statusRecorder{ResponseWriter: rec, status: http.StatusOK}
	sr.WriteHeader(http.StatusCreated)

	if sr.status != 201 {
		t.Errorf("expected status 201, got %d", sr.status)
	}
	if rec.Code != 201 {
		t.Errorf("expected underlying recorder code 201, got %d", rec.Code)
	}
}
