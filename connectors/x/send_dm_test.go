package x

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSendDM_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/dm_conversations/with/user123/messages" {
			t.Errorf("path = %s, want /dm_conversations/with/user123/messages", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]string
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if reqBody["text"] != "Hello there!" {
			t.Errorf("text = %q, want %q", reqBody["text"], "Hello there!")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"dm_conversation_id": "conv123",
				"dm_event_id":        "event456",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.send_dm"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.send_dm",
		Parameters:  json.RawMessage(`{"recipient_id":"user123","text":"Hello there!"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["dm_conversation_id"] != "conv123" {
		t.Errorf("dm_conversation_id = %v, want conv123", data["dm_conversation_id"])
	}
}

func TestSendDM_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["x.send_dm"]

	longText := make([]byte, 10001)
	for i := range longText {
		longText[i] = 'a'
	}

	tests := []struct {
		name   string
		params string
	}{
		{"missing recipient_id", `{"text":"hello"}`},
		{"missing text", `{"recipient_id":"user123"}`},
		{"text too long", `{"recipient_id":"user123","text":"` + string(longText) + `"}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "x.send_dm",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("Execute() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestSendDM_ForbiddenError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"detail": "You are not permitted to send DMs to this user",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.send_dm"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.send_dm",
		Parameters:  json.RawMessage(`{"recipient_id":"user123","text":"hello"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}
