package protonmail

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSearchEmails_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchEmailsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.search_emails",
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

func TestSearchEmails_NoCriteria(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchEmailsAction{conn: conn}

	params, _ := json.Marshal(map[string]any{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.search_emails",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for no search criteria")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchEmails_InvalidSinceDate(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchEmailsAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"since": "not-a-date",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.search_emails",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid since date")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchEmails_InvalidBeforeDate(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchEmailsAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"subject": "test",
		"before":  "not-a-date",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.search_emails",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid before date")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchEmails_LimitTooHigh(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchEmailsAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"subject": "test",
		"limit":   100,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.search_emails",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for limit too high")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchEmailsParams_Defaults(t *testing.T) {
	t.Parallel()

	p := &searchEmailsParams{Subject: "test"}
	if err := p.validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Folder != "INBOX" {
		t.Errorf("expected default folder 'INBOX', got %q", p.Folder)
	}
	if p.Limit != 10 {
		t.Errorf("expected default limit 10, got %d", p.Limit)
	}
}
