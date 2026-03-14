package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestArchiveEmail_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/gmail/v1/users/me/messages/msg-123/modify" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ya29.test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}

		var body gmailModifyRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if len(body.RemoveLabelIDs) != 1 || body.RemoveLabelIDs[0] != "INBOX" {
			t.Errorf("expected RemoveLabelIDs=[INBOX], got %v", body.RemoveLabelIDs)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gmailModifyResponse{
			ID:       "msg-123",
			ThreadID: "thread-456",
			LabelIDs: []string{"IMPORTANT"},
		})
	}))
	defer srv.Close()

	conn := newGmailForTest(srv.Client(), srv.URL)
	action := &archiveEmailAction{conn: conn}

	params, _ := json.Marshal(archiveEmailParams{MessageID: "msg-123"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.archive_email",
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
	if data["id"] != "msg-123" {
		t.Errorf("expected id 'msg-123', got %q", data["id"])
	}
	if data["thread_id"] != "thread-456" {
		t.Errorf("expected thread_id 'thread-456', got %q", data["thread_id"])
	}
}

func TestArchiveEmail_MissingMessageID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &archiveEmailAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.archive_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing message_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestArchiveEmail_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &archiveEmailAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.archive_email",
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

func TestArchiveEmail_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    401,
				"message": "Invalid Credentials",
			},
		})
	}))
	defer srv.Close()

	conn := newGmailForTest(srv.Client(), srv.URL)
	action := &archiveEmailAction{conn: conn}

	params, _ := json.Marshal(archiveEmailParams{MessageID: "msg-123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.archive_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T (%v)", err, err)
	}
}
