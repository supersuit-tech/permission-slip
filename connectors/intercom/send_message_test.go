package intercom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSendMessage_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/messages" {
			t.Errorf("expected path /messages, got %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if body["message_type"] != "inapp" {
			t.Errorf("expected message_type inapp, got %v", body["message_type"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(outboundMessageResponse{
			Type:        "user_message",
			ID:          "msg_001",
			MessageType: "inapp",
			Body:        "Hello there!",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendMessageAction{conn: conn}

	params, _ := json.Marshal(sendMessageParams{
		Body:        "Hello there!",
		FromAdminID: "admin_1",
		ToContactID: "contact_1",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.send_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data outboundMessageResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.ID != "msg_001" {
		t.Errorf("expected id msg_001, got %q", data.ID)
	}
}

func TestSendMessage_MissingBody(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendMessageAction{conn: conn}

	params, _ := json.Marshal(sendMessageParams{
		FromAdminID: "admin_1",
		ToContactID: "contact_1",
	})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.send_message",
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

func TestSendMessage_EmailMissingSubject(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendMessageAction{conn: conn}

	params, _ := json.Marshal(sendMessageParams{
		Body:        "Hello!",
		MessageType: "email",
		FromAdminID: "admin_1",
		ToContactID: "contact_1",
	})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.send_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for email without subject")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
