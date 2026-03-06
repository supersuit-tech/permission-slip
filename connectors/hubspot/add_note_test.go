package hubspot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestAddNote_Success(t *testing.T) {
	t.Parallel()

	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)

		if r.URL.Path == "/crm/v3/objects/notes" && r.Method == http.MethodPost {
			var body hubspotObjectRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			if body.Properties["hs_note_body"] != "Called and left a voicemail" {
				t.Errorf("expected note body, got %q", body.Properties["hs_note_body"])
			}
			if body.Properties["hs_timestamp"] == "" {
				t.Error("expected hs_timestamp to be set")
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{
				"id":         "801",
				"properties": body.Properties,
			})
			return
		}

		// Association call
		if strings.Contains(r.URL.Path, "/associations/") {
			if r.Method != http.MethodPut {
				t.Errorf("expected PUT for association, got %s", r.Method)
			}
			expectedPath := "/crm/v3/objects/notes/801/associations/contacts/501/note_to_contact"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &addNoteAction{conn: conn}

	params, _ := json.Marshal(addNoteParams{
		ObjectType: "contact",
		ObjectID:   "501",
		Body:       "Called and left a voicemail",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.add_note",
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
	if data["note_id"] != "801" {
		t.Errorf("expected note_id 801, got %q", data["note_id"])
	}
	if data["object_type"] != "contact" {
		t.Errorf("expected object_type contact, got %q", data["object_type"])
	}

	if len(calls) != 2 {
		t.Errorf("expected 2 API calls (create + associate), got %d: %v", len(calls), calls)
	}
}

func TestAddNote_DealAssociation(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/crm/v3/objects/notes" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{"id": "802", "properties": map[string]string{}})
			return
		}
		expectedPath := "/crm/v3/objects/notes/802/associations/deals/601/note_to_deal"
		if r.URL.Path != expectedPath {
			t.Errorf("expected association path %s, got %s", expectedPath, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &addNoteAction{conn: conn}

	params, _ := json.Marshal(addNoteParams{
		ObjectType: "deal",
		ObjectID:   "601",
		Body:       "Deal note",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.add_note",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddNote_MissingObjectType(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addNoteAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"object_id": "501",
		"body":      "Note",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.add_note",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing object_type")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAddNote_InvalidObjectType(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addNoteAction{conn: conn}

	params, _ := json.Marshal(addNoteParams{
		ObjectType: "company",
		ObjectID:   "501",
		Body:       "Note",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.add_note",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid object_type")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAddNote_MissingBody(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addNoteAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"object_type": "contact",
		"object_id":   "501",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.add_note",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing body")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAddNote_AssociationFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/crm/v3/objects/notes" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{"id": "803", "properties": map[string]string{}})
			return
		}
		// Association fails.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "error",
			"category": "OBJECT_NOT_FOUND",
			"message":  "Contact not found",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &addNoteAction{conn: conn}

	params, _ := json.Marshal(addNoteParams{
		ObjectType: "contact",
		ObjectID:   "999",
		Body:       "Note",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.add_note",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for association failure")
	}
}
