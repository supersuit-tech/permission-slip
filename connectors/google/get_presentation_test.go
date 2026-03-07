package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetPresentation_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/presentations/pres-abc-123" {
			t.Errorf("expected path /v1/presentations/pres-abc-123, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ya29.test-access-token-123" {
			t.Errorf("unexpected Authorization header: %s", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(slidesGetResponse{
			PresentationID: "pres-abc-123",
			Title:          "Q1 Review",
			Slides: []slidePage{
				{ObjectID: "slide-001"},
				{ObjectID: "slide-002"},
				{ObjectID: "slide-003"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &getPresentationAction{conn: conn}

	params, _ := json.Marshal(getPresentationParams{
		PresentationID: "pres-abc-123",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.get_presentation",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		PresentationID string   `json:"presentation_id"`
		Title          string   `json:"title"`
		URL            string   `json:"url"`
		SlideCount     int      `json:"slide_count"`
		Slides         []string `json:"slides"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.PresentationID != "pres-abc-123" {
		t.Errorf("expected presentation_id 'pres-abc-123', got %q", data.PresentationID)
	}
	if data.Title != "Q1 Review" {
		t.Errorf("expected title 'Q1 Review', got %q", data.Title)
	}
	if len(data.Slides) != 3 {
		t.Fatalf("expected 3 slides, got %d", len(data.Slides))
	}
	if data.Slides[0] != "slide-001" {
		t.Errorf("expected first slide 'slide-001', got %q", data.Slides[0])
	}
	if data.SlideCount != 3 {
		t.Errorf("expected slide_count 3, got %d", data.SlideCount)
	}
	wantURL := "https://docs.google.com/presentation/d/pres-abc-123/edit"
	if data.URL != wantURL {
		t.Errorf("expected url %q, got %q", wantURL, data.URL)
	}
}

func TestGetPresentation_EmptySlides(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(slidesGetResponse{
			PresentationID: "pres-empty",
			Title:          "Empty Deck",
			Slides:         nil,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &getPresentationAction{conn: conn}

	params, _ := json.Marshal(getPresentationParams{
		PresentationID: "pres-empty",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.get_presentation",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		SlideCount int      `json:"slide_count"`
		Slides     []string `json:"slides"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Slides) != 0 {
		t.Errorf("expected 0 slides, got %d", len(data.Slides))
	}
	if data.SlideCount != 0 {
		t.Errorf("expected slide_count 0, got %d", data.SlideCount)
	}
}

func TestGetPresentation_MissingPresentationID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getPresentationAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.get_presentation",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing presentation_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetPresentation_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getPresentationAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.get_presentation",
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

func TestGetPresentation_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 401, "message": "Invalid Credentials"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &getPresentationAction{conn: conn}

	params, _ := json.Marshal(getPresentationParams{
		PresentationID: "pres-abc-123",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.get_presentation",
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

func TestGetPresentation_PresentationIDURLEncoded(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantPath := "/v1/presentations/pres%2Fwith%2Fslashes"
		if r.URL.RawPath != wantPath {
			t.Errorf("expected raw path %q, got %q", wantPath, r.URL.RawPath)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(slidesGetResponse{
			PresentationID: "pres/with/slashes",
			Title:          "Special ID",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &getPresentationAction{conn: conn}

	params, _ := json.Marshal(getPresentationParams{
		PresentationID: "pres/with/slashes",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.get_presentation",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
