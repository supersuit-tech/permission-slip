package google

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// replyTestHandler returns an httptest.HandlerFunc that serves the message
// fetch and the send endpoint for send_email_reply tests.
func replyTestHandler(t *testing.T, threadID, from, subject, messageID string, sentCapture *gmailSendReplyRequest) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/messages/") {
			json.NewEncoder(w).Encode(map[string]any{
				"id":       "msg001",
				"threadId": threadID,
				"payload": map[string]any{
					"headers": []map[string]string{
						{"name": "From", "value": from},
						{"name": "Subject", "value": subject},
						{"name": "Message-ID", "value": messageID},
					},
				},
			})
			return
		}
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/messages/send") {
			if sentCapture != nil {
				json.NewDecoder(r.Body).Decode(sentCapture)
			}
			json.NewEncoder(w).Encode(map[string]string{
				"id":       "reply001",
				"threadId": threadID,
			})
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}
}

func TestSendEmailReply_Success(t *testing.T) {
	var sent gmailSendReplyRequest
	srv := httptest.NewServer(replyTestHandler(t,
		"thread1", "sender@example.com", "Hello", "<msg-id-1@example.com>", &sent,
	))
	defer srv.Close()

	conn := newGmailForTest(srv.Client(), srv.URL)
	action := &sendEmailReplyAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"thread_id":  "thread1",
		"message_id": "msg001",
		"body":       "Thanks for reaching out.",
	})
	result, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out map[string]string
	if err := json.Unmarshal(result.Data, &out); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if out["id"] != "reply001" {
		t.Errorf("expected id reply001, got %s", out["id"])
	}
	if out["thread_id"] != "thread1" {
		t.Errorf("expected thread_id thread1, got %s", out["thread_id"])
	}
	if out["to"] != "sender@example.com" {
		t.Errorf("expected to sender@example.com, got %s", out["to"])
	}
	if out["subject"] != "Re: Hello" {
		t.Errorf("expected subject 'Re: Hello', got %s", out["subject"])
	}
	if sent.ThreadID != "thread1" {
		t.Errorf("expected threadId thread1 in sent body, got %s", sent.ThreadID)
	}
}

func TestSendEmailReply_SubjectPrefixed(t *testing.T) {
	// When original subject already starts with "Re:", it should not be doubled.
	var sent gmailSendReplyRequest
	srv := httptest.NewServer(replyTestHandler(t,
		"thread2", "other@example.com", "Re: Existing Thread", "<mid@x.com>", &sent,
	))
	defer srv.Close()

	conn := newGmailForTest(srv.Client(), srv.URL)
	action := &sendEmailReplyAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"thread_id":  "thread2",
		"message_id": "msg002",
		"body":       "Still relevant.",
	})
	result, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out map[string]string
	json.Unmarshal(result.Data, &out)
	if out["subject"] != "Re: Existing Thread" {
		t.Errorf("expected 'Re: Existing Thread', got %s", out["subject"])
	}

	// Decode the raw message and verify Subject header is not doubled.
	raw, err := base64.RawURLEncoding.DecodeString(sent.Raw)
	if err != nil {
		t.Fatalf("decode raw: %v", err)
	}
	rawStr := string(raw)
	if !strings.Contains(rawStr, "Subject: Re: Existing Thread\r\n") {
		t.Errorf("expected exact Subject header in raw message, got:\n%s", rawStr)
	}
	if !strings.Contains(rawStr, "multipart/alternative") {
		t.Error("expected multipart for default html reply")
	}
}

func TestSendEmailReply_ThreadIDMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			// Return a message that belongs to a different thread.
			json.NewEncoder(w).Encode(map[string]any{
				"id":       "msg003",
				"threadId": "different-thread",
				"payload": map[string]any{
					"headers": []map[string]string{
						{"name": "From", "value": "x@example.com"},
					},
				},
			})
			return
		}
		t.Error("should not reach send endpoint")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	conn := newGmailForTest(srv.Client(), srv.URL)
	action := &sendEmailReplyAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"thread_id":  "expected-thread",
		"message_id": "msg003",
		"body":       "Hello.",
	})
	_, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for thread_id mismatch")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestSendEmailReply_MissingFrom(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return a message with no From header.
		json.NewEncoder(w).Encode(map[string]any{
			"id":       "msg004",
			"threadId": "thread4",
			"payload": map[string]any{
				"headers": []map[string]string{
					{"name": "Subject", "value": "No Sender"},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newGmailForTest(srv.Client(), srv.URL)
	action := &sendEmailReplyAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"thread_id":  "thread4",
		"message_id": "msg004",
		"body":       "Hello.",
	})
	_, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing From header")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestSendEmailReply_MissingThreadID(t *testing.T) {
	conn := newGmailForTest(nil, "http://unused")
	action := &sendEmailReplyAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"message_id": "msg001",
		"body":       "Hello.",
	})
	_, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing thread_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestSendEmailReply_HTMLFalsePlaintext(t *testing.T) {
	var sent gmailSendReplyRequest
	srv := httptest.NewServer(replyTestHandler(t,
		"threadPlain", "sender@example.com", "Sub", "<mid@x.com>", &sent,
	))
	defer srv.Close()

	conn := newGmailForTest(srv.Client(), srv.URL)
	action := &sendEmailReplyAction{conn: conn}
	htmlFalse := false
	params, _ := json.Marshal(map[string]any{
		"thread_id":  "threadPlain",
		"message_id": "msgPlain",
		"body":       "<b>not html</b>",
		"html":       htmlFalse,
	})
	_, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(sent.Raw)
	if err != nil {
		t.Fatalf("decode raw: %v", err)
	}
	rawStr := string(raw)
	if strings.Contains(rawStr, "multipart/alternative") {
		t.Error("expected single part for html=false reply")
	}
	if !strings.Contains(rawStr, "<b>not html</b>") {
		t.Error("expected literal angle brackets in plaintext reply body")
	}
}

func TestSendEmailReply_InvalidJSON(t *testing.T) {
	conn := newGmailForTest(nil, "http://unused")
	action := &sendEmailReplyAction{conn: conn}

	_, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  []byte(`{bad json`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
