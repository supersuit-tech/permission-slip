package salesforce

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestAddNote_Success(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		calls.Add(1)
		switch r.URL.Path {
		case "/services/data/v62.0/sobjects/ContentNote/":
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode ContentNote request: %v", err)
			}
			if body["Title"] != "Meeting Notes" {
				t.Errorf("expected Title 'Meeting Notes', got %q", body["Title"])
			}
			if body["Content"] == "" {
				t.Error("expected non-empty base64 Content")
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(sfContentNoteResponse{ID: "069xx0000000001", Success: true})

		case "/services/data/v62.0/sobjects/ContentDocumentLink/":
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode ContentDocumentLink request: %v", err)
			}
			if body["ContentDocumentId"] != "069xx0000000001" {
				t.Errorf("expected ContentDocumentId '069xx0000000001', got %q", body["ContentDocumentId"])
			}
			if body["LinkedEntityId"] != "001xx0000000001" {
				t.Errorf("expected LinkedEntityId '001xx0000000001', got %q", body["LinkedEntityId"])
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(sfCreateResponse{ID: "06Axx0000000001", Success: true})

		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &addNoteAction{conn: conn}

	params, _ := json.Marshal(addNoteParams{
		ParentID: "001xx0000000001",
		Title:    "Meeting Notes",
		Body:     "Discussed Q3 targets.",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.add_note",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["note_id"] != "069xx0000000001" {
		t.Errorf("expected note_id '069xx0000000001', got %v", data["note_id"])
	}
	if data["link_id"] != "06Axx0000000001" {
		t.Errorf("expected link_id '06Axx0000000001', got %v", data["link_id"])
	}
	if data["success"] != true {
		t.Errorf("expected success true, got %v", data["success"])
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("expected 2 API calls (ContentNote + ContentDocumentLink), got %d", got)
	}
}

func TestAddNote_LinkingFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/services/data/v62.0/sobjects/ContentNote/":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(sfContentNoteResponse{ID: "069xx0000000002", Success: true})

		case "/services/data/v62.0/sobjects/ContentDocumentLink/":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode([]sfAPIError{{ErrorCode: "INVALID_CROSS_REFERENCE_KEY", Message: "Invalid parent ID"}})
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &addNoteAction{conn: conn}

	params, _ := json.Marshal(addNoteParams{
		ParentID: "001xx000000BAD1",
		Title:    "Test Note",
		Body:     "body",
	})

	// Should return partial success (note created, link failed).
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.add_note",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("expected partial success, got error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["note_id"] != "069xx0000000002" {
		t.Errorf("expected note_id '069xx0000000002', got %v", data["note_id"])
	}
	if data["success"] != false {
		t.Errorf("expected success false for partial failure, got %v", data["success"])
	}
	if data["warning"] == nil || data["warning"] == "" {
		t.Error("expected warning about linking failure")
	}
}

func TestAddNote_MissingParentID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addNoteAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"title": "Test"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.add_note",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing parent_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAddNote_MissingTitle(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addNoteAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"parent_id": "001xx0000000001"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.add_note",
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

func TestAddNote_EmptyBody(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/services/data/v62.0/sobjects/ContentNote/":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(sfContentNoteResponse{ID: "069xx0000000003", Success: true})
		case "/services/data/v62.0/sobjects/ContentDocumentLink/":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(sfCreateResponse{ID: "06Axx0000000002", Success: true})
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &addNoteAction{conn: conn}

	params, _ := json.Marshal(addNoteParams{
		ParentID: "001xx0000000001",
		Title:    "Empty Note",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.add_note",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["success"] != true {
		t.Errorf("expected success true, got %v", data["success"])
	}
}

func TestAddNote_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addNoteAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.add_note",
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

func TestAddNote_InvalidParentID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addNoteAction{conn: conn}

	params, _ := json.Marshal(addNoteParams{
		ParentID: "abc",
		Title:    "Test",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.add_note",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid parent_id format")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
