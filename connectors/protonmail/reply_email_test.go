package protonmail

import (
	"context"
	"encoding/json"
	"fmt"
	"net/smtp"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestReplyEmail_Success(t *testing.T) {
	orig := resolveSourceForReply
	t.Cleanup(func() { resolveSourceForReply = orig })

	resolveSourceForReply = func(_ context.Context, _ *ProtonMailConnector, _ connectors.Credentials, folder string, uid uint32, _ connectors.MailboxUIDValidityStore) (*replySource, error) {
		if folder != "INBOX" || uid != 42 {
			t.Fatalf("folder=%q uid=%d", folder, uid)
		}
		return &replySource{
			From:            []string{"alice@example.com"},
			Subject:         "Weekly Update",
			MessageIDHeader: "<msg-42@example.com>",
		}, nil
	}

	var capturedTo []string
	var capturedMsg []byte

	conn := New()
	action := &replyEmailAction{
		conn: conn,
		sendFunc: func(_ string, _ smtp.Auth, _ string, to []string, msg []byte) error {
			capturedTo = to
			capturedMsg = msg
			return nil
		},
	}

	params, _ := json.Marshal(replyEmailParams{
		InReplyToMessageID: 42,
		Body:               "Thanks for the update!",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.reply_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedTo) != 1 || capturedTo[0] != "alice@example.com" {
		t.Errorf("expected to [alice@example.com], got %v", capturedTo)
	}

	msgStr := string(capturedMsg)
	for _, want := range []string{
		"To: alice@example.com",
		"Subject: Re: Weekly Update",
		"In-Reply-To: <msg-42@example.com>",
		"References: <msg-42@example.com>",
		"Thanks for the update!",
	} {
		if !strings.Contains(msgStr, want) {
			t.Errorf("expected message to contain %q, got %q", want, msgStr)
		}
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["status"] != "sent" {
		t.Errorf("status = %v, want sent", data["status"])
	}
}

func TestReplyEmail_ExplicitToAndSubject(t *testing.T) {
	orig := resolveSourceForReply
	t.Cleanup(func() { resolveSourceForReply = orig })

	resolveSourceForReply = func(_ context.Context, _ *ProtonMailConnector, _ connectors.Credentials, _ string, _ uint32, _ connectors.MailboxUIDValidityStore) (*replySource, error) {
		return &replySource{
			From:            []string{"alice@example.com"},
			Subject:         "Original",
			MessageIDHeader: "<orig@example.com>",
		}, nil
	}

	var capturedMsg []byte
	conn := New()
	action := &replyEmailAction{
		conn: conn,
		sendFunc: func(_ string, _ smtp.Auth, _ string, _ []string, msg []byte) error {
			capturedMsg = msg
			return nil
		},
	}

	params, _ := json.Marshal(replyEmailParams{
		InReplyToMessageID: 1,
		To:                 []string{"bob@example.com"},
		Subject:            "Custom subject",
		Body:               "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.reply_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msgStr := string(capturedMsg)
	if !strings.Contains(msgStr, "To: bob@example.com") {
		t.Errorf("expected explicit to, got %q", msgStr)
	}
	if !strings.Contains(msgStr, "Subject: Custom subject") {
		t.Errorf("expected explicit subject, got %q", msgStr)
	}
}

func TestReplyEmail_MissingInReplyToMessageID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &replyEmailAction{conn: conn}
	params, _ := json.Marshal(map[string]any{"body": "Hi"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.reply_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestReplyEmail_SourceNotFound(t *testing.T) {
	orig := resolveSourceForReply
	t.Cleanup(func() { resolveSourceForReply = orig })

	resolveSourceForReply = func(_ context.Context, _ *ProtonMailConnector, _ connectors.Credentials, _ string, _ uint32, _ connectors.MailboxUIDValidityStore) (*replySource, error) {
		return nil, &connectors.ValidationError{Message: "message uid 99 not found in folder \"INBOX\""}
	}

	conn := New()
	action := &replyEmailAction{conn: conn}
	params, _ := json.Marshal(replyEmailParams{InReplyToMessageID: 99, Body: "Hi"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.reply_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestReplyEmail_MissingMessageIDHeader(t *testing.T) {
	orig := resolveSourceForReply
	t.Cleanup(func() { resolveSourceForReply = orig })

	resolveSourceForReply = func(_ context.Context, _ *ProtonMailConnector, _ connectors.Credentials, _ string, _ uint32, _ connectors.MailboxUIDValidityStore) (*replySource, error) {
		return &replySource{
			From:            []string{"alice@example.com"},
			Subject:         "Hello",
			MessageIDHeader: "",
		}, nil
	}

	conn := New()
	action := &replyEmailAction{conn: conn}
	params, _ := json.Marshal(replyEmailParams{InReplyToMessageID: 1, Body: "Hi"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.reply_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T", err)
	}
}

func TestReplyEmail_HeaderInjectionPrevented(t *testing.T) {
	orig := resolveSourceForReply
	t.Cleanup(func() { resolveSourceForReply = orig })

	resolveSourceForReply = func(_ context.Context, _ *ProtonMailConnector, _ connectors.Credentials, _ string, _ uint32, _ connectors.MailboxUIDValidityStore) (*replySource, error) {
		return &replySource{
			From:            []string{"alice@example.com"},
			Subject:         "Safe",
			MessageIDHeader: "<safe@example.com>\r\nBcc: attacker@evil.com",
		}, nil
	}

	var capturedMsg []byte
	conn := New()
	action := &replyEmailAction{
		conn: conn,
		sendFunc: func(_ string, _ smtp.Auth, _ string, _ []string, msg []byte) error {
			capturedMsg = msg
			return nil
		},
	}

	params, _ := json.Marshal(replyEmailParams{InReplyToMessageID: 1, Body: "Hi"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.reply_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msgStr := string(capturedMsg)
	lines := strings.Split(msgStr, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Bcc:") {
			t.Error("header injection: injected Bcc header found as separate line")
		}
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "In-Reply-To:") && strings.Contains(line, "\r") {
			t.Error("header injection: CR found in In-Reply-To line")
		}
	}
}

func TestReplyEmail_SMTPError(t *testing.T) {
	orig := resolveSourceForReply
	t.Cleanup(func() { resolveSourceForReply = orig })

	resolveSourceForReply = func(_ context.Context, _ *ProtonMailConnector, _ connectors.Credentials, _ string, _ uint32, _ connectors.MailboxUIDValidityStore) (*replySource, error) {
		return &replySource{
			From:            []string{"alice@example.com"},
			Subject:         "Hello",
			MessageIDHeader: "<x@example.com>",
		}, nil
	}

	conn := New()
	action := &replyEmailAction{
		conn: conn,
		sendFunc: func(_ string, _ smtp.Auth, _ string, _ []string, _ []byte) error {
			return fmt.Errorf("connection refused")
		},
	}

	params, _ := json.Marshal(replyEmailParams{InReplyToMessageID: 1, Body: "Hi"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.reply_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected SMTP error")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T", err)
	}
}

func TestBuildReplyMessage(t *testing.T) {
	t.Parallel()

	msg := buildReplyMessage(
		"sender@proton.me",
		[]string{"alice@example.com"},
		[]string{"bob@example.com"},
		nil,
		"Re: Hello",
		"Reply body",
		"text/plain",
		"<orig@example.com>",
	)

	msgStr := string(msg)
	checks := []string{
		"From: sender@proton.me",
		"To: alice@example.com",
		"Cc: bob@example.com",
		"Subject: Re: Hello",
		"In-Reply-To: <orig@example.com>",
		"References: <orig@example.com>",
		"Reply body",
	}
	for _, c := range checks {
		if !strings.Contains(msgStr, c) {
			t.Errorf("expected %q in message", c)
		}
	}
}
