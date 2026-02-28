package notify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func sendGridTestApproval() Approval {
	return Approval{
		ApprovalID:  "appr_test123",
		AgentID:     42,
		AgentName:   "Deploy Bot",
		Action:      json.RawMessage(`{"type":"email.send","version":"1","parameters":{"to":"bob@example.com"}}`),
		Context:     json.RawMessage(`{"description":"Send deployment notification","risk_level":"low"}`),
		ApprovalURL: "https://app.example.com/approve/appr_test123",
		ExpiresAt:   time.Now().Add(5 * time.Minute),
		CreatedAt:   time.Now(),
	}
}

func sendGridTestRecipient() Recipient {
	email := "alice@example.com"
	return Recipient{
		UserID:   "user-001",
		Username: "alice",
		Email:    &email,
	}
}

func TestSendGrid_Success(t *testing.T) {
	t.Parallel()

	var receivedBody []byte
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	sender := &SendGridSender{
		apiKey: "SG.test-key",
		from:   "noreply@example.com",
		client: server.Client(),
	}

	err := sender.sendToURL(context.Background(), sendGridTestApproval(), sendGridTestRecipient(), server.URL)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if receivedAuth != "Bearer SG.test-key" {
		t.Errorf("expected Bearer auth header, got: %s", receivedAuth)
	}

	var payload sendGridPayload
	if err := json.Unmarshal(receivedBody, &payload); err != nil {
		t.Fatalf("failed to unmarshal request body: %v", err)
	}

	if payload.From.Email != "noreply@example.com" {
		t.Errorf("expected from=noreply@example.com, got: %s", payload.From.Email)
	}
	if len(payload.Personalizations) == 0 || len(payload.Personalizations[0].To) == 0 {
		t.Fatal("expected at least one recipient")
	}
	if payload.Personalizations[0].To[0].Email != "alice@example.com" {
		t.Errorf("expected to=alice@example.com, got: %s", payload.Personalizations[0].To[0].Email)
	}
	if payload.Subject == "" {
		t.Error("expected non-empty subject")
	}
	if len(payload.Content) != 2 {
		t.Errorf("expected 2 content parts (plain+html), got: %d", len(payload.Content))
	}
}

func TestSendGrid_SubjectIncludesActionType(t *testing.T) {
	t.Parallel()

	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	sender := &SendGridSender{
		apiKey: "SG.test-key",
		from:   "noreply@example.com",
		client: server.Client(),
	}

	err := sender.sendToURL(context.Background(), sendGridTestApproval(), sendGridTestRecipient(), server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload sendGridPayload
	if err := json.Unmarshal(receivedBody, &payload); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if payload.Subject != "Approval needed: email.send" {
		t.Errorf("expected subject containing action type, got: %s", payload.Subject)
	}
}

func TestSendGrid_APIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"errors":[{"message":"invalid API key"}]}`))
	}))
	defer server.Close()

	sender := &SendGridSender{
		apiKey: "bad-key",
		from:   "noreply@example.com",
		client: server.Client(),
	}

	err := sender.sendToURL(context.Background(), sendGridTestApproval(), sendGridTestRecipient(), server.URL)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestSendGrid_NoEmail(t *testing.T) {
	t.Parallel()

	sender := NewSendGridSender("SG.test", "noreply@example.com")

	recipient := Recipient{UserID: "user-001", Username: "alice"}
	err := sender.Send(context.Background(), sendGridTestApproval(), recipient)
	if err == nil {
		t.Fatal("expected error for nil email")
	}
}

func TestSendGrid_EmptyEmail(t *testing.T) {
	t.Parallel()

	sender := NewSendGridSender("SG.test", "noreply@example.com")

	empty := ""
	recipient := Recipient{UserID: "user-001", Username: "alice", Email: &empty}
	err := sender.Send(context.Background(), sendGridTestApproval(), recipient)
	if err == nil {
		t.Fatal("expected error for empty email")
	}
}

func TestSendGrid_ContextCancelled(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	sender := &SendGridSender{
		apiKey: "SG.test",
		from:   "noreply@example.com",
		client: server.Client(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := sender.sendToURL(ctx, sendGridTestApproval(), sendGridTestRecipient(), server.URL)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestSendGrid_Name(t *testing.T) {
	t.Parallel()
	sender := NewSendGridSender("key", "from@example.com")
	if sender.Name() != "email" {
		t.Errorf("expected name=email, got: %s", sender.Name())
	}
}

func TestSendGrid_SendUsesBaseURLOverride(t *testing.T) {
	t.Parallel()

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	sender := &SendGridSender{
		apiKey:  "SG.test",
		from:    "noreply@example.com",
		client:  server.Client(),
		baseURL: server.URL,
	}

	err := sender.Send(context.Background(), sendGridTestApproval(), sendGridTestRecipient())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected test server to be called via baseURL override")
	}
}
