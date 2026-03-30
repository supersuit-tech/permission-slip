package zoom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestAddRegistrant_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/meetings/98765432100/registrants" {
			t.Errorf("expected path /meetings/98765432100/registrants, got %s", r.URL.Path)
		}

		var body addRegistrantRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body.Email != "alice@example.com" {
			t.Errorf("expected email 'alice@example.com', got %q", body.Email)
		}
		if body.FirstName != "Alice" {
			t.Errorf("expected first_name 'Alice', got %q", body.FirstName)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(addRegistrantResponse{
			RegistrantID: "reg-abc123",
			JoinURL:      "https://zoom.us/j/98765432100?pwd=abc",
			Topic:        "Product Demo",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &addRegistrantAction{conn: conn}

	params, _ := json.Marshal(addRegistrantParams{
		MeetingID: "98765432100",
		Email:     "alice@example.com",
		FirstName: "Alice",
		LastName:  "Smith",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.add_registrant",
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
	if data["registrant_id"] != "reg-abc123" {
		t.Errorf("expected registrant_id 'reg-abc123', got %q", data["registrant_id"])
	}
	if data["join_url"] == "" {
		t.Error("expected non-empty join_url")
	}
}

func TestAddRegistrant_MissingEmail(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addRegistrantAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"meeting_id": "98765432100",
		"first_name": "Alice",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.add_registrant",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestAddRegistrant_InvalidEmail(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addRegistrantAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"meeting_id": "98765432100",
		"email":      "not-an-email",
		"first_name": "Alice",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.add_registrant",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}
