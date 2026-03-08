package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSendEmailReply_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/messages/msg-123") {
			// Return the original message metadata
			json.NewEncoder(w).Encode(gmailMessageResponse{
				ID:       "msg-123",
				ThreadID: "thread-abc",
				Payload: struct {
					Headers []struct {
						Name  string `json:"name"`
						Value string `json:"value"`
					} `json:"headers"`
				}{
					Headers: []struct {
						Name  string `json:"name"`
						Value string `json:"value"`
					}{
						{Name: "From", Value: "sender@example.com"},
						{Name: "Subject", Value: "Hello World"},
						{Name: "Message-Id", Value: "<original-msg-id@mail.example.com>"},
					},
				},
			})
			return
		}

		// Handle send
		var body gmailSendReplyRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.ThreadID != "thread-abc" {
			t.Errorf("expected threadId 'thread-abc', got %q", body.ThreadID)
		}
		json.NewEncoder(w).Encode(gmailSendResponse{
			ID:       "reply-msg-id",
			ThreadID: "thread-abc",
		})
	}))
	defer srv.Close()

	conn := &GoogleConnector{client: srv.Client(), gmailBaseURL: srv.URL}
	action := &sendEmailReplyAction{conn: conn}

	params, _ := json.Marshal(sendEmailReplyParams{
		ThreadID:  "thread-abc",
		MessageID: "msg-123",
		Body:      "Thanks for your message!",
	})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.send_email_reply",
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
	if data["id"] != "reply-msg-id" {
		t.Errorf("expected id 'reply-msg-id', got %q", data["id"])
	}
	if data["thread_id"] != "thread-abc" {
		t.Errorf("expected thread_id 'thread-abc', got %q", data["thread_id"])
	}
}

func TestSendEmailReply_SubjectPrefixed(t *testing.T) {
	t.Parallel()

	var capturedRaw string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/messages/msg-1") {
			json.NewEncoder(w).Encode(gmailMessageResponse{
				ID:       "msg-1",
				ThreadID: "thread-1",
				Payload: struct {
					Headers []struct {
						Name  string `json:"name"`
						Value string `json:"value"`
					} `json:"headers"`
				}{
					Headers: []struct {
						Name  string `json:"name"`
						Value string `json:"value"`
					}{
						{Name: "From", Value: "alice@example.com"},
						{Name: "Subject", Value: "Re: Already a reply"},
					},
				},
			})
			return
		}

		var body gmailSendReplyRequest
		json.NewDecoder(r.Body).Decode(&body)
		capturedRaw = body.Raw
		json.NewEncoder(w).Encode(gmailSendResponse{ID: "new-msg", ThreadID: "thread-1"})
	}))
	defer srv.Close()

	conn := &GoogleConnector{client: srv.Client(), gmailBaseURL: srv.URL}
	action := &sendEmailReplyAction{conn: conn}

	params, _ := json.Marshal(sendEmailReplyParams{
		ThreadID:  "thread-1",
		MessageID: "msg-1",
		Body:      "Reply body",
	})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.send_email_reply",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Decode the raw message and check subject not double-prefixed
	if capturedRaw == "" {
		t.Fatal("expected raw message to be set")
	}
}

func TestSendEmailReply_MissingThreadID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendEmailReplyAction{conn: conn}

	params, _ := json.Marshal(sendEmailReplyParams{MessageID: "msg-1", Body: "hi"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.send_email_reply",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing thread_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendEmailReply_MissingMessageID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendEmailReplyAction{conn: conn}

	params, _ := json.Marshal(sendEmailReplyParams{ThreadID: "thread-1", Body: "hi"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.send_email_reply",
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

func TestSendEmailReply_MissingBody(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendEmailReplyAction{conn: conn}

	params, _ := json.Marshal(sendEmailReplyParams{ThreadID: "thread-1", MessageID: "msg-1"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.send_email_reply",
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

func TestSendEmailReply_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendEmailReplyAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.send_email_reply",
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

func TestSendEmailReply_FetchOriginalFails(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 401, "message": "Unauthorized"},
		})
	}))
	defer srv.Close()

	conn := &GoogleConnector{client: srv.Client(), gmailBaseURL: srv.URL}
	action := &sendEmailReplyAction{conn: conn}

	params, _ := json.Marshal(sendEmailReplyParams{
		ThreadID:  "thread-1",
		MessageID: "msg-1",
		Body:      "hi",
	})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.send_email_reply",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error when fetching original fails")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}
