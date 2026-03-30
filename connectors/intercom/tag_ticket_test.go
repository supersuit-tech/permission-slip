package intercom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestTagTicket_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/tags" {
			t.Errorf("expected path /tags, got %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			return
		}
		if body["name"] != "vip" {
			t.Errorf("expected tag name 'vip', got %v", body["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tag{
			Type: "tag",
			ID:   "tag-1",
			Name: "vip",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &tagTicketAction{conn: conn}

	params, _ := json.Marshal(tagTicketParams{
		TagName:  "vip",
		TicketID: "42",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.tag_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data tag
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.Name != "vip" {
		t.Errorf("expected tag name 'vip', got %q", data.Name)
	}
}

func TestTagTicket_MissingTagName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &tagTicketAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"ticket_id": "42"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.tag_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing tag_name")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestTagTicket_MissingTicketID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &tagTicketAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"tag_name": "vip"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.tag_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing ticket_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
