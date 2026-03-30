package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreatePresentation_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/presentations" {
			t.Errorf("expected path /v1/presentations, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ya29.test-access-token-123" {
			t.Errorf("unexpected Authorization header: %s", got)
		}

		var body slidesCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Title != "Q1 Review" {
			t.Errorf("expected title 'Q1 Review', got %q", body.Title)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(slidesCreateResponse{
			PresentationID: "pres-abc-123",
			Title:          "Q1 Review",
		})
	}))
	defer srv.Close()

	conn := newForTestWithSlides(srv.Client(), "", "", srv.URL)
	action := &createPresentationAction{conn: conn}

	params, _ := json.Marshal(createPresentationParams{
		Title: "Q1 Review",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_presentation",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["presentation_id"] != "pres-abc-123" {
		t.Errorf("expected presentation_id 'pres-abc-123', got %q", data["presentation_id"])
	}
	if data["title"] != "Q1 Review" {
		t.Errorf("expected title 'Q1 Review', got %q", data["title"])
	}
	wantURL := "https://docs.google.com/presentation/d/pres-abc-123/edit"
	if data["url"] != wantURL {
		t.Errorf("expected url %q, got %q", wantURL, data["url"])
	}
}

func TestCreatePresentation_MissingTitle(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPresentationAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_presentation",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing title")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreatePresentation_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPresentationAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_presentation",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreatePresentation_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 401, "message": "Invalid Credentials"},
		})
	}))
	defer srv.Close()

	conn := newForTestWithSlides(srv.Client(), "", "", srv.URL)
	action := &createPresentationAction{conn: conn}

	params, _ := json.Marshal(createPresentationParams{
		Title: "Test Deck",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_presentation",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}

func TestCreatePresentation_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 429, "message": "Rate Limit Exceeded"},
		})
	}))
	defer srv.Close()

	conn := newForTestWithSlides(srv.Client(), "", "", srv.URL)
	action := &createPresentationAction{conn: conn}

	params, _ := json.Marshal(createPresentationParams{
		Title: "Test Deck",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_presentation",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for rate limit")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
}
