package linkedin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSendMessage_Success(t *testing.T) {
	t.Parallel()

	var gotBody sendMessageRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/messages":
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if got := r.Header.Get("LinkedIn-Version"); got != linkedInVersion {
				t.Errorf("expected LinkedIn-Version %q, got %q", linkedInVersion, got)
			}
			if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
				t.Errorf("failed to decode request body: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &sendMessageAction{conn: conn}

	params, _ := json.Marshal(sendMessageParams{
		RecipientURN: "urn:li:person:123456",
		Subject:      "Hello",
		Body:         "This is a test message.",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.send_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(gotBody.Recipients) != 1 || gotBody.Recipients[0] != "urn:li:person:123456" {
		t.Errorf("unexpected recipients: %v", gotBody.Recipients)
	}
	if gotBody.Subject != "Hello" {
		t.Errorf("expected subject 'Hello', got %q", gotBody.Subject)
	}
	if gotBody.Body != "This is a test message." {
		t.Errorf("expected body 'This is a test message.', got %q", gotBody.Body)
	}
	if gotBody.MessageType != "MEMBER_TO_MEMBER" {
		t.Errorf("expected messageType 'MEMBER_TO_MEMBER', got %q", gotBody.MessageType)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["status"] != "sent" {
		t.Errorf("expected status 'sent', got %q", data["status"])
	}
}

func TestSendMessage_MissingRecipientURN(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"body": "Hello"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.send_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing recipient_urn")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendMessage_InvalidRecipientURN(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendMessageAction{conn: conn}

	params, _ := json.Marshal(sendMessageParams{
		RecipientURN: "not-a-valid-urn",
		Body:         "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.send_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid recipient_urn")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendMessage_ShareURNRejected(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendMessageAction{conn: conn}

	// share URNs look valid as generic URNs but must be rejected for messaging
	params, _ := json.Marshal(sendMessageParams{
		RecipientURN: "urn:li:share:123456",
		Body:         "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.send_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for share URN used as recipient_urn")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendMessage_MissingBody(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"recipient_urn": "urn:li:person:123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.send_message",
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

func TestSendMessage_SubjectTooLong(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendMessageAction{conn: conn}

	params, _ := json.Marshal(sendMessageParams{
		RecipientURN: "urn:li:person:123",
		Subject:      strings.Repeat("s", maxMessageSubjectLen+1),
		Body:         "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.send_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for subject too long")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendMessage_BodyTooLong(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendMessageAction{conn: conn}

	params, _ := json.Marshal(sendMessageParams{
		RecipientURN: "urn:li:person:123",
		Body:         strings.Repeat("a", maxMessageBodyLen+1),
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.send_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for body too long")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendMessage_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendMessageAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.send_message",
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
