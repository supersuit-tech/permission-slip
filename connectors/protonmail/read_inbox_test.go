package protonmail

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestReadInbox_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &readInboxAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.read_inbox",
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

func TestReadInbox_LimitTooHigh(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &readInboxAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"limit": 100,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.read_inbox",
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

func TestReadInboxParams_Defaults(t *testing.T) {
	t.Parallel()

	p := &readInboxParams{}
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
