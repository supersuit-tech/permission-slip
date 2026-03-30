package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestAddSlide_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/presentations/pres-abc-123:batchUpdate" {
			t.Errorf("expected path /v1/presentations/pres-abc-123:batchUpdate, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ya29.test-access-token-123" {
			t.Errorf("unexpected Authorization header: %s", got)
		}

		var body slidesBatchUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if len(body.Requests) != 1 {
			t.Fatalf("expected 1 request, got %d", len(body.Requests))
		}
		cs := body.Requests[0].CreateSlide
		if cs == nil {
			t.Fatal("expected createSlide request")
		}
		if cs.SlideLayoutReference == nil || cs.SlideLayoutReference.PredefinedLayout != "BLANK" {
			t.Errorf("expected BLANK layout, got %+v", cs.SlideLayoutReference)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(slidesBatchUpdateResponse{
			Replies: []slidesBatchReply{
				{CreateSlide: &createSlideReply{ObjectID: "slide-new-001"}},
			},
		})
	}))
	defer srv.Close()

	conn := newForTestWithSlides(srv.Client(), "", "", srv.URL)
	action := &addSlideAction{conn: conn}

	params, _ := json.Marshal(addSlideParams{
		PresentationID: "pres-abc-123",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.add_slide",
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
	if data["slide_id"] != "slide-new-001" {
		t.Errorf("expected slide_id 'slide-new-001', got %q", data["slide_id"])
	}
	if data["presentation_id"] != "pres-abc-123" {
		t.Errorf("expected presentation_id 'pres-abc-123', got %q", data["presentation_id"])
	}
}

func TestAddSlide_WithLayout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body slidesBatchUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		cs := body.Requests[0].CreateSlide
		if cs.SlideLayoutReference.PredefinedLayout != "TITLE_AND_BODY" {
			t.Errorf("expected TITLE_AND_BODY layout, got %q", cs.SlideLayoutReference.PredefinedLayout)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(slidesBatchUpdateResponse{
			Replies: []slidesBatchReply{
				{CreateSlide: &createSlideReply{ObjectID: "slide-new-002"}},
			},
		})
	}))
	defer srv.Close()

	conn := newForTestWithSlides(srv.Client(), "", "", srv.URL)
	action := &addSlideAction{conn: conn}

	params, _ := json.Marshal(addSlideParams{
		PresentationID: "pres-abc-123",
		Layout:         "TITLE_AND_BODY",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.add_slide",
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
	if data["slide_id"] != "slide-new-002" {
		t.Errorf("expected slide_id 'slide-new-002', got %q", data["slide_id"])
	}
}

func TestAddSlide_WithInsertionIndex(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body slidesBatchUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		cs := body.Requests[0].CreateSlide
		if cs.InsertionIndex == nil || *cs.InsertionIndex != 2 {
			t.Errorf("expected insertion_index 2, got %v", cs.InsertionIndex)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(slidesBatchUpdateResponse{
			Replies: []slidesBatchReply{
				{CreateSlide: &createSlideReply{ObjectID: "slide-new-003"}},
			},
		})
	}))
	defer srv.Close()

	conn := newForTestWithSlides(srv.Client(), "", "", srv.URL)
	action := &addSlideAction{conn: conn}

	idx := 2
	params, _ := json.Marshal(addSlideParams{
		PresentationID: "pres-abc-123",
		InsertionIndex: &idx,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.add_slide",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddSlide_MissingPresentationID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addSlideAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.add_slide",
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

func TestAddSlide_InvalidLayout(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addSlideAction{conn: conn}

	params, _ := json.Marshal(addSlideParams{
		PresentationID: "pres-abc-123",
		Layout:         "INVALID_LAYOUT",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.add_slide",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid layout")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAddSlide_NegativeInsertionIndex(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addSlideAction{conn: conn}

	idx := -1
	params, _ := json.Marshal(addSlideParams{
		PresentationID: "pres-abc-123",
		InsertionIndex: &idx,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.add_slide",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for negative insertion_index")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAddSlide_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addSlideAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.add_slide",
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

func TestAddSlide_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 401, "message": "Invalid Credentials"},
		})
	}))
	defer srv.Close()

	conn := newForTestWithSlides(srv.Client(), "", "", srv.URL)
	action := &addSlideAction{conn: conn}

	params, _ := json.Marshal(addSlideParams{
		PresentationID: "pres-abc-123",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.add_slide",
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
