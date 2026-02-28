package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSON_Success(t *testing.T) {
	t.Parallel()
	body := `{"name":"test","value":42}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	var dst struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	if err := DecodeJSON(r, &dst); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dst.Name != "test" {
		t.Errorf("expected name 'test', got %q", dst.Name)
	}
	if dst.Value != 42 {
		t.Errorf("expected value 42, got %d", dst.Value)
	}
}

func TestDecodeJSON_MissingContentType_DefaultsToJSON(t *testing.T) {
	t.Parallel()
	body := `{"name":"test"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	// No Content-Type header set — should default to JSON

	var dst struct {
		Name string `json:"name"`
	}
	if err := DecodeJSON(r, &dst); err != nil {
		t.Fatalf("expected missing Content-Type to default to JSON, got error: %v", err)
	}
	if dst.Name != "test" {
		t.Errorf("expected name 'test', got %q", dst.Name)
	}
}

func TestDecodeJSON_WrongContentType(t *testing.T) {
	t.Parallel()
	body := `{"name":"test"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "text/plain")

	var dst struct{}
	err := DecodeJSON(r, &dst)
	if err != errContentType {
		t.Errorf("expected errContentType, got %v", err)
	}
}

func TestDecodeJSON_ContentTypeWithCharset(t *testing.T) {
	t.Parallel()
	body := `{"name":"test"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json; charset=utf-8")

	var dst struct {
		Name string `json:"name"`
	}

	if err := DecodeJSON(r, &dst); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dst.Name != "test" {
		t.Errorf("expected name 'test', got %q", dst.Name)
	}
}

func TestDecodeJSON_MalformedJSON(t *testing.T) {
	t.Parallel()
	body := `{invalid json`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	var dst struct{}
	err := DecodeJSON(r, &dst)
	if err != errMalformedJSON {
		t.Errorf("expected errMalformedJSON, got %v", err)
	}
}

func TestDecodeJSON_EmptyBody(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	r.Header.Set("Content-Type", "application/json")

	var dst struct{}
	err := DecodeJSON(r, &dst)
	if err != errMalformedJSON {
		t.Errorf("expected errMalformedJSON, got %v", err)
	}
}

func TestDecodeJSON_IgnoresUnknownFields(t *testing.T) {
	t.Parallel()
	body := `{"name":"test","unknown":"field"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	var dst struct {
		Name string `json:"name"`
	}

	if err := DecodeJSON(r, &dst); err != nil {
		t.Fatalf("expected unknown fields to be ignored, got error: %v", err)
	}
	if dst.Name != "test" {
		t.Errorf("expected name 'test', got %q", dst.Name)
	}
}

func TestDecodeJSON_TrailingContent(t *testing.T) {
	t.Parallel()
	body := `{"name":"test"}{"extra":"object"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	var dst struct {
		Name string `json:"name"`
	}

	err := DecodeJSON(r, &dst)
	if err != errMalformedJSON {
		t.Errorf("expected errMalformedJSON for trailing content, got %v", err)
	}
}

func TestDecodeJSON_BodyTooLarge(t *testing.T) {
	t.Parallel()
	// Create a body larger than MaxRequestBodySize (1 MB).
	bigBody := `{"data":"` + strings.Repeat("x", MaxRequestBodySize+1) + `"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(bigBody))
	r.Header.Set("Content-Type", "application/json")

	var dst struct {
		Data string `json:"data"`
	}

	err := DecodeJSON(r, &dst)
	if err != errBodyTooLarge {
		t.Errorf("expected errBodyTooLarge, got %v", err)
	}
}

func TestDecodeJSONOrReject_Success(t *testing.T) {
	t.Parallel()
	body := `{"name":"test"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	var dst struct {
		Name string `json:"name"`
	}

	ok := DecodeJSONOrReject(w, r, &dst)
	if !ok {
		t.Fatal("expected DecodeJSONOrReject to return true")
	}
	if dst.Name != "test" {
		t.Errorf("expected name 'test', got %q", dst.Name)
	}
}

func TestDecodeJSONOrReject_BadContentType(t *testing.T) {
	t.Parallel()
	body := `{"name":"test"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	var dst struct{}
	ok := DecodeJSONOrReject(w, r, &dst)
	if ok {
		t.Fatal("expected DecodeJSONOrReject to return false")
	}
	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected status 415, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != ErrInvalidRequest {
		t.Errorf("expected error code %q, got %q", ErrInvalidRequest, errResp.Error.Code)
	}
	if errResp.Error.Message != "Content-Type must be application/json" {
		t.Errorf("expected message 'Content-Type must be application/json', got %q", errResp.Error.Message)
	}
}

func TestDecodeJSONOrReject_MalformedBody(t *testing.T) {
	t.Parallel()
	body := `not json`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	var dst struct{}
	ok := DecodeJSONOrReject(w, r, &dst)
	if ok {
		t.Fatal("expected DecodeJSONOrReject to return false")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != ErrInvalidRequest {
		t.Errorf("expected error code %q, got %q", ErrInvalidRequest, errResp.Error.Code)
	}
	if errResp.Error.Message != "Malformed or invalid JSON" {
		t.Errorf("expected message 'Malformed or invalid JSON', got %q", errResp.Error.Message)
	}
}

func TestDecodeJSONOrReject_BodyTooLarge(t *testing.T) {
	t.Parallel()
	bigBody := `{"data":"` + strings.Repeat("x", MaxRequestBodySize+1) + `"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(bigBody))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	var dst struct {
		Data string `json:"data"`
	}

	ok := DecodeJSONOrReject(w, r, &dst)
	if ok {
		t.Fatal("expected DecodeJSONOrReject to return false")
	}
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status 413, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != ErrInvalidRequest {
		t.Errorf("expected error code %q, got %q", ErrInvalidRequest, errResp.Error.Code)
	}
	if errResp.Error.Message != "Request body too large" {
		t.Errorf("expected message 'Request body too large', got %q", errResp.Error.Message)
	}
}

func TestRespondJSON(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()

	payload := map[string]string{"status": "ok"}
	RespondJSON(w, http.StatusOK, payload)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", ct)
	}

	var parsed map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if parsed["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", parsed["status"])
	}
}

func TestRespondJSON_CustomStatus(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()

	payload := map[string]int{"id": 1}
	RespondJSON(w, http.StatusCreated, payload)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}
}
